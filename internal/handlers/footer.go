package handlers

import (
	"net/mail"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"github.com/example/shafran/internal/models"
)

// FooterHandler manages footer settings endpoints.
type FooterHandler struct {
	db *gorm.DB
}

// NewFooterHandler constructs FooterHandler.
func NewFooterHandler(db *gorm.DB) *FooterHandler {
	return &FooterHandler{db: db}
}

const (
	defaultAddress              = "Toshkent, Mustaqillik ko'chasi, 1-uy"
	defaultPhone                = "+998 99 999 99 99"
	defaultEmail                = "info@shafran.uz"
	defaultWorkingHours         = "Dushanba - Shanba: 09:00 - 18:00"
	defaultWorkingHoursUz       = "Dushanba - Shanba: 09:00 - 18:00"
	defaultWorkingHoursRu       = "Пн - Сб: 09:00 - 18:00"
	defaultWorkingHoursEn       = "Mon - Sat: 09:00 - 18:00"
	defaultWorkingHoursTitleUz  = "Ish vaqti"
	defaultWorkingHoursTitleRu  = "Рабочее время"
	defaultWorkingHoursTitleEn  = "Working hours"
	defaultSubscribeTitleUz     = "Eksklyuziv takliflar va so'nggi yangiliklarni olish uchun obuna bo'ling!"
	defaultSubscribeTitleRu     = "Подпишитесь, чтобы получать эксклюзивные предложения и последние новости!"
	defaultSubscribeTitleEn     = "Subscribe to receive exclusive offers and the latest news!"
	defaultCopyrightTextUz      = "© 2026 SHAFRAN. Barcha huquqlar himoyalangan."
	defaultCopyrightTextRu      = "© 2026 SHAFRAN. Все права защищены."
	defaultCopyrightTextEn      = "© 2026 SHAFRAN. All Rights Reserved."
)

func applyFooterDefaults(settings *models.FooterSettings) {
	if settings == nil {
		return
	}
	if strings.TrimSpace(settings.Address) == "" {
		settings.Address = defaultAddress
	}
	if strings.TrimSpace(settings.Phone) == "" {
		settings.Phone = defaultPhone
	}
	if strings.TrimSpace(settings.Email) == "" {
		settings.Email = defaultEmail
	}
	if strings.TrimSpace(settings.WorkingHours) == "" {
		settings.WorkingHours = defaultWorkingHours
	}
	if strings.TrimSpace(settings.WorkingHoursUz) == "" {
		settings.WorkingHoursUz = defaultWorkingHoursUz
	}
	if strings.TrimSpace(settings.WorkingHoursRu) == "" {
		settings.WorkingHoursRu = defaultWorkingHoursRu
	}
	if strings.TrimSpace(settings.WorkingHoursEn) == "" {
		settings.WorkingHoursEn = defaultWorkingHoursEn
	}
	if strings.TrimSpace(settings.WorkingHoursTitleUz) == "" {
		settings.WorkingHoursTitleUz = defaultWorkingHoursTitleUz
	}
	if strings.TrimSpace(settings.WorkingHoursTitleRu) == "" {
		settings.WorkingHoursTitleRu = defaultWorkingHoursTitleRu
	}
	if strings.TrimSpace(settings.WorkingHoursTitleEn) == "" {
		settings.WorkingHoursTitleEn = defaultWorkingHoursTitleEn
	}
	if strings.TrimSpace(settings.SubscribeTitleUz) == "" {
		settings.SubscribeTitleUz = defaultSubscribeTitleUz
	}
	if strings.TrimSpace(settings.SubscribeTitleRu) == "" {
		settings.SubscribeTitleRu = defaultSubscribeTitleRu
	}
	if strings.TrimSpace(settings.SubscribeTitleEn) == "" {
		settings.SubscribeTitleEn = defaultSubscribeTitleEn
	}
	if strings.TrimSpace(settings.CopyrightTextUz) == "" {
		settings.CopyrightTextUz = defaultCopyrightTextUz
	}
	if strings.TrimSpace(settings.CopyrightTextRu) == "" {
		settings.CopyrightTextRu = defaultCopyrightTextRu
	}
	if strings.TrimSpace(settings.CopyrightTextEn) == "" {
		settings.CopyrightTextEn = defaultCopyrightTextEn
	}
}

func validateFooterSettings(input *models.FooterSettings) error {
	if input == nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if strings.TrimSpace(input.Email) != "" {
		if _, err := mail.ParseAddress(input.Email); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid email format")
		}
	}
	return nil
}

// GetFooter returns the current footer settings (public endpoint).
func (h *FooterHandler) GetFooter(c *fiber.Ctx) error {
	var settings models.FooterSettings
	result := h.db.First(&settings)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// Return default settings for first load
			defaults := models.FooterSettings{}
			applyFooterDefaults(&defaults)
			return c.JSON(fiber.Map{
				"success": true,
				"data":    defaults,
			})
		}
		return result.Error
	}

	applyFooterDefaults(&settings)
	return c.JSON(fiber.Map{
		"success": true,
		"data":    settings,
	})
}

// UpdateFooter creates or updates footer settings (admin endpoint).
func (h *FooterHandler) UpdateFooter(c *fiber.Ctx) error {
	var input models.FooterSettings
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if err := validateFooterSettings(&input); err != nil {
		return err
	}

	var existing models.FooterSettings
	result := h.db.First(&existing)

	if result.Error == gorm.ErrRecordNotFound {
		// Create new
		if err := h.db.Create(&input).Error; err != nil {
			return err
		}
		return c.JSON(fiber.Map{
			"success": true,
			"data":    input,
		})
	} else if result.Error != nil {
		return result.Error
	}

	// Update existing record explicitly to avoid overwriting immutable fields
	// like created_at with zero values from client payload.
	existing.Address = input.Address
	existing.Phone = input.Phone
	existing.Phone2 = input.Phone2
	existing.Email = input.Email

	existing.WorkingHours = input.WorkingHours
	existing.WorkingHoursUz = input.WorkingHoursUz
	existing.WorkingHoursRu = input.WorkingHoursRu
	existing.WorkingHoursEn = input.WorkingHoursEn
	existing.WorkingHoursTitleUz = input.WorkingHoursTitleUz
	existing.WorkingHoursTitleRu = input.WorkingHoursTitleRu
	existing.WorkingHoursTitleEn = input.WorkingHoursTitleEn

	existing.Telegram = input.Telegram
	existing.Instagram = input.Instagram
	existing.Facebook = input.Facebook
	existing.Youtube = input.Youtube
	existing.Twitter = input.Twitter
	existing.TikTok = input.TikTok

	existing.TelegramEnabled = input.TelegramEnabled
	existing.InstagramEnabled = input.InstagramEnabled
	existing.FacebookEnabled = input.FacebookEnabled
	existing.YoutubeEnabled = input.YoutubeEnabled
	existing.TwitterEnabled = input.TwitterEnabled
	existing.TikTokEnabled = input.TikTokEnabled

	existing.SubscribeTitleUz = input.SubscribeTitleUz
	existing.SubscribeTitleRu = input.SubscribeTitleRu
	existing.SubscribeTitleEn = input.SubscribeTitleEn
	existing.CopyrightTextUz = input.CopyrightTextUz
	existing.CopyrightTextRu = input.CopyrightTextRu
	existing.CopyrightTextEn = input.CopyrightTextEn
	existing.CopyrightText = input.CopyrightText

	if err := h.db.Save(&existing).Error; err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    existing,
	})
}
