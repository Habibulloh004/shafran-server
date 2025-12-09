package middleware

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/example/shafran/internal/services"
)

type paymeRequestID struct {
	ID any `json:"id"`
}

// PaymeAuthMiddleware validates the Payme Authorization header.
func PaymeAuthMiddleware(merchantKey string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var reqID paymeRequestID
		_ = json.Unmarshal(c.Body(), &reqID)

		authHeader := c.Get("Authorization")
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 {
			return writePaymeAuthError(c, reqID.ID)
		}

		token := parts[1]
		decoded, err := base64.StdEncoding.DecodeString(token)
		if err != nil {
			return writePaymeAuthError(c, reqID.ID)
		}

		if !strings.Contains(string(decoded), merchantKey) {
			return writePaymeAuthError(c, reqID.ID)
		}

		return c.Next()
	}
}

func writePaymeAuthError(c *fiber.Ctx, id any) error {
	info := services.PaymeErrorInvalidAuthorization
	return c.JSON(fiber.Map{
		"error": fiber.Map{
			"code": info.Code,
			"message": fiber.Map{
				"uz": info.Message["uz"],
				"ru": info.Message["ru"],
				"en": info.Message["en"],
			},
			"data": nil,
		},
		"id": id,
	})
}

