package utils

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// Pagination holds pagination parameters.
type Pagination struct {
	Page   int
	Limit  int
	Offset int
}

// ParsePagination reads page and limit query params with sane defaults.
func ParsePagination(c *fiber.Ctx) Pagination {
	page := parseInt(c.Query("page", "1"), 1)
	limit := parseInt(c.Query("limit", "20"), 20)
	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}

	return Pagination{
		Page:   page,
		Limit:  limit,
		Offset: (page - 1) * limit,
	}
}

func parseInt(value string, fallback int) int {
	if parsed, err := strconv.Atoi(value); err == nil {
		return parsed
	}
	return fallback
}

