package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/example/shafran/internal/config"
	"github.com/example/shafran/internal/utils"
)

const userContextKey = "currentUserID"

// AuthMiddleware validates JWT tokens and loads the authenticated user ID into context.
func AuthMiddleware(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "missing authorization header")
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid authorization header")
		}

		userID, err := utils.ParseToken(cfg.JWTSecret, parts[1])
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
		}

		c.Locals(userContextKey, userID)
		return c.Next()
	}
}

// GetCurrentUserID extracts the authenticated user ID from context.
func GetCurrentUserID(c *fiber.Ctx) (uuid.UUID, bool) {
	value := c.Locals(userContextKey)
	if value == nil {
		return uuid.Nil, false
	}

	if id, ok := value.(uuid.UUID); ok {
		return id, true
	}

	return uuid.Nil, false
}
