package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"github.com/example/shafran/internal/config"
	"github.com/example/shafran/internal/models"
	"github.com/example/shafran/internal/services"
	"github.com/example/shafran/internal/utils"
)

// PasswordResetHandler manages forgot-password endpoints.
type PasswordResetHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

// NewPasswordResetHandler constructs a PasswordResetHandler.
func NewPasswordResetHandler(db *gorm.DB, cfg *config.Config) *PasswordResetHandler {
	return &PasswordResetHandler{db: db, cfg: cfg}
}

type forgotPasswordRequest struct {
	Phone string `json:"phone"`
}

// ForgotPassword initiates the password-reset flow: validates user, generates
// a 6-digit code, sends it via Plum, and returns a reset token.
func (h *PasswordResetHandler) ForgotPassword(c *fiber.Ctx) error {
	var req forgotPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if req.Phone == "" {
		return fiber.NewError(fiber.StatusBadRequest, "phone is required")
	}

	// Check user exists.
	var user models.User
	if err := h.db.Where("phone = ?", req.Phone).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "user not found")
		}
		return err
	}

	// Generate a 6-digit code.
	code, err := generateResetCode()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to generate code")
	}

	// Generate a secure reset token.
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to generate token")
	}
	resetToken := hex.EncodeToString(tokenBytes)

	// Try sending code via Plum; if disabled/fails, still store the code (fallback).
	var sessionID string
	plumCfg := services.LoadPlumConfig()
	if plumCfg.Enabled {
		sid, err := services.PlumVerifyPhone(req.Phone)
		if err != nil {
			// Log but don't fail — store code for manual/SMS fallback.
			fmt.Printf("plum verify phone failed: %v\n", err)
		} else {
			sessionID = sid
		}
	}

	// Expire any previous unused reset tokens for this phone.
	h.db.Model(&models.PasswordResetToken{}).
		Where("phone = ? AND used_at IS NULL", req.Phone).
		Update("expires_at", time.Now())

	// Create new reset token.
	resetRecord := models.PasswordResetToken{
		Phone:     req.Phone,
		Token:     resetToken,
		Code:      code,
		SessionID: sessionID,
		ExpiresAt: time.Now().Add(10 * time.Minute),
		Verified:  false,
	}
	if err := h.db.Create(&resetRecord).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create reset token")
	}

	return c.JSON(fiber.Map{
		"success":    true,
		"token":      resetToken,
		"session_id": sessionID,
		"code":       code,
	})
}

type verifyResetCodeRequest struct {
	Token     string `json:"token"`
	Code      string `json:"code"`
	SessionID string `json:"session_id"`
}

// VerifyResetCode verifies the code submitted by the user.
func (h *PasswordResetHandler) VerifyResetCode(c *fiber.Ctx) error {
	var req verifyResetCodeRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if req.Token == "" || req.Code == "" {
		return fiber.NewError(fiber.StatusBadRequest, "token and code are required")
	}

	var record models.PasswordResetToken
	if err := h.db.Where("token = ?", req.Token).First(&record).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "invalid reset token")
		}
		return err
	}

	if record.UsedAt != nil {
		return fiber.NewError(fiber.StatusBadRequest, "token already used")
	}

	if record.ExpiresAt.Before(time.Now()) {
		return fiber.NewError(fiber.StatusBadRequest, "token expired")
	}

	// Verify code: try Plum first, then fallback to local code.
	plumCfg := services.LoadPlumConfig()
	verified := false

	if plumCfg.Enabled && req.SessionID != "" {
		ok, err := services.PlumConfirmCode(req.SessionID, req.Code)
		if err == nil && ok {
			verified = true
		}
	}

	// Fallback: compare with stored code.
	if !verified && record.Code == req.Code {
		verified = true
	}

	if !verified {
		return fiber.NewError(fiber.StatusBadRequest, "invalid verification code")
	}

	record.Verified = true
	if err := h.db.Save(&record).Error; err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success":  true,
		"verified": true,
		"token":    record.Token,
	})
}

type resetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// ResetPassword updates the user's password after successful code verification.
func (h *PasswordResetHandler) ResetPassword(c *fiber.Ctx) error {
	var req resetPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if req.Token == "" || req.NewPassword == "" {
		return fiber.NewError(fiber.StatusBadRequest, "token and new_password are required")
	}

	if len(req.NewPassword) < 6 {
		return fiber.NewError(fiber.StatusBadRequest, "password must be at least 6 characters")
	}

	var record models.PasswordResetToken
	if err := h.db.Where("token = ?", req.Token).First(&record).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "invalid reset token")
		}
		return err
	}

	if record.UsedAt != nil {
		return fiber.NewError(fiber.StatusBadRequest, "token already used")
	}

	if record.ExpiresAt.Before(time.Now()) {
		return fiber.NewError(fiber.StatusBadRequest, "token expired")
	}

	if !record.Verified {
		return fiber.NewError(fiber.StatusBadRequest, "code not verified yet")
	}

	// Hash new password.
	hash, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to hash password")
	}

	// Update user password.
	if err := h.db.Model(&models.User{}).
		Where("phone = ?", record.Phone).
		Update("password_hash", hash).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update password")
	}

	// Mark token as used.
	now := time.Now()
	record.UsedAt = &now
	h.db.Save(&record)

	return c.JSON(fiber.Map{
		"success": true,
		"message": "password updated successfully",
	})
}

func generateResetCode() (string, error) {
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
