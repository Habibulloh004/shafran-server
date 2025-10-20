package handlers

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"github.com/example/shafran/internal/config"
	"github.com/example/shafran/internal/models"
	"github.com/example/shafran/internal/utils"
)

// AuthHandler bundles dependencies for authentication endpoints.
type AuthHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

// NewAuthHandler constructs an AuthHandler.
func NewAuthHandler(db *gorm.DB, cfg *config.Config) *AuthHandler {
	return &AuthHandler{db: db, cfg: cfg}
}

type registerRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone"`
	Password  string `json:"password"`
}

// Register creates a new user account.
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req registerRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if req.Phone == "" || req.Password == "" || req.FirstName == "" {
		return fiber.NewError(fiber.StatusBadRequest, "missing required fields")
	}

	var existing models.User
	if err := h.db.Where("phone = ?", req.Phone).First(&existing).Error; err == nil {
		return fiber.NewError(fiber.StatusConflict, "user already exists")
	} else if err != gorm.ErrRecordNotFound {
		return err
	}

	passwordHash, err := utils.HashPassword(req.Password)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to hash password")
	}

	user := models.User{
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Phone:        req.Phone,
		DisplayName:  fmt.Sprintf("%s %s", req.FirstName, req.LastName),
		PasswordHash: passwordHash,
		IsVerified:   false,
	}

	if err := h.db.Create(&user).Error; err != nil {
		return err
	}

	code, err := generateVerificationCode()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to generate verification code")
	}

	verification := models.SMSVerification{
		Phone:     req.Phone,
		Code:      code,
		ExpiresAt: time.Now().Add(10 * time.Minute),
		Verified:  false,
	}

	if err := h.db.Create(&verification).Error; err != nil {
		return err
	}

	token, err := utils.GenerateToken(h.cfg.JWTSecret, user.ID, h.cfg.TokenExpires)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to generate token")
	}

	respUser := map[string]interface{}{
		"id":           user.ID,
		"first_name":   user.FirstName,
		"last_name":    user.LastName,
		"phone":        user.Phone,
		"display_name": user.DisplayName,
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"user":    respUser,
		"token":   token,
	})
}

type loginRequest struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

// Login authenticates an existing user.
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	var user models.User
	if err := h.db.Where("phone = ?", req.Phone).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid credentials")
		}
		return err
	}

	if !utils.CheckPassword(user.PasswordHash, req.Password) {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid credentials")
	}

	token, err := utils.GenerateToken(h.cfg.JWTSecret, user.ID, h.cfg.TokenExpires)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to generate token")
	}

	respUser := map[string]interface{}{
		"id":           user.ID,
		"display_name": user.DisplayName,
		"phone":        user.Phone,
	}

	return c.JSON(fiber.Map{
		"success": true,
		"user":    respUser,
		"token":   token,
	})
}

type verifyRequest struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
}

// Verify handles SMS code validation.
func (h *AuthHandler) Verify(c *fiber.Ctx) error {
	var req verifyRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	var verification models.SMSVerification
	err := h.db.Where("phone = ?", req.Phone).
		Order("created_at desc").
		First(&verification).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "verification code not found")
		}
		return err
	}

	if verification.Code != req.Code {
		return fiber.NewError(fiber.StatusBadRequest, "invalid verification code")
	}

	if verification.ExpiresAt.Before(time.Now()) {
		return fiber.NewError(fiber.StatusBadRequest, "verification code expired")
	}

	verification.Verified = true
	now := time.Now()
	verification.UsedAt = &now
	if err := h.db.Save(&verification).Error; err != nil {
		return err
	}

	if err := h.db.Model(&models.User{}).Where("phone = ?", req.Phone).
		Update("is_verified", true).Error; err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success":  true,
		"verified": true,
	})
}

func generateVerificationCode() (string, error) {
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
