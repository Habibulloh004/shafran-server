package handlers

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"github.com/example/shafran/internal/models"
	"github.com/example/shafran/internal/utils"
)

// AdminHandler manages admin-only endpoints.
type AdminHandler struct {
	db *gorm.DB
}

// NewAdminHandler constructs AdminHandler.
func NewAdminHandler(db *gorm.DB) *AdminHandler {
	return &AdminHandler{db: db}
}

// DashboardStats returns aggregate statistics for the admin dashboard.
func (h *AdminHandler) DashboardStats(c *fiber.Ctx) error {
	var totalUsers int64
	if err := h.db.Model(&models.User{}).Count(&totalUsers).Error; err != nil {
		return err
	}

	var totalOrders int64
	if err := h.db.Model(&models.Order{}).Count(&totalOrders).Error; err != nil {
		return err
	}

	// Orders by status
	type statusCount struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	var statusCounts []statusCount
	if err := h.db.Model(&models.Order{}).
		Select("status, count(*) as count").
		Group("status").
		Scan(&statusCounts).Error; err != nil {
		return err
	}

	ordersByStatus := make(map[string]int64)
	for _, sc := range statusCounts {
		ordersByStatus[sc.Status] = sc.Count
	}

	// Total revenue (sum of total_amount for non-cancelled orders)
	var totalRevenue float64
	if err := h.db.Model(&models.Order{}).
		Where("status != ?", "cancelled").
		Select("COALESCE(SUM(total_amount), 0)").
		Scan(&totalRevenue).Error; err != nil {
		return err
	}

	// Today's revenue
	var todayRevenue float64
	if err := h.db.Model(&models.Order{}).
		Where("status != ? AND placed_at::date = CURRENT_DATE", "cancelled").
		Select("COALESCE(SUM(total_amount), 0)").
		Scan(&todayRevenue).Error; err != nil {
		return err
	}

	// Banners count
	var totalBanners int64
	if err := h.db.Model(&models.Banner{}).Count(&totalBanners).Error; err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"total_users":      totalUsers,
			"total_orders":     totalOrders,
			"total_banners":    totalBanners,
			"total_revenue":    totalRevenue,
			"today_revenue":    todayRevenue,
			"orders_by_status": ordersByStatus,
		},
	})
}

// ListAllOrders returns all orders with pagination, filtering, and user info.
func (h *AdminHandler) ListAllOrders(c *fiber.Ctx) error {
	pg := utils.ParsePagination(c)
	query := h.db.Model(&models.Order{})

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	if search := c.Query("search"); search != "" {
		query = query.Where(
			"order_number ILIKE ? OR delivery_address_line ILIKE ?",
			"%"+search+"%", "%"+search+"%",
		)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return err
	}

	var orders []models.Order
	if err := query.Preload("Items").Preload("User").
		Order("placed_at desc").
		Limit(pg.Limit).Offset(pg.Offset).
		Find(&orders).Error; err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    orders,
		"pagination": fiber.Map{
			"current_page":   pg.Page,
			"items_per_page": pg.Limit,
			"total_items":    total,
		},
	})
}

// ListAllUsers returns all registered users with pagination and search.
func (h *AdminHandler) ListAllUsers(c *fiber.Ctx) error {
	pg := utils.ParsePagination(c)
	query := h.db.Model(&models.User{})

	if search := c.Query("search"); search != "" {
		query = query.Where(
			"first_name ILIKE ? OR last_name ILIKE ? OR phone ILIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%",
		)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return err
	}

	// Select specific fields to avoid exposing password hash
	var users []models.User
	if err := query.Select("id, first_name, last_name, phone, display_name, is_verified, created_at, updated_at").
		Order("created_at desc").
		Limit(pg.Limit).Offset(pg.Offset).
		Find(&users).Error; err != nil {
		return err
	}

	// Enrich users with order counts and total spent
	type userStats struct {
		UserID     string  `json:"user_id"`
		OrderCount int64   `json:"order_count"`
		TotalSpent float64 `json:"total_spent"`
	}

	var stats []userStats
	h.db.Model(&models.Order{}).
		Select("user_id, count(*) as order_count, COALESCE(SUM(total_amount), 0) as total_spent").
		Group("user_id").
		Scan(&stats)

	statsMap := make(map[string]userStats)
	for _, s := range stats {
		statsMap[s.UserID] = s
	}

	type userResponse struct {
		models.User
		OrderCount int64   `json:"order_count"`
		TotalSpent float64 `json:"total_spent"`
	}

	result := make([]userResponse, len(users))
	for i, u := range users {
		result[i] = userResponse{User: u}
		if s, ok := statsMap[u.ID.String()]; ok {
			result[i].OrderCount = s.OrderCount
			result[i].TotalSpent = s.TotalSpent
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result,
		"pagination": fiber.Map{
			"current_page":   pg.Page,
			"items_per_page": pg.Limit,
			"total_items":    total,
		},
	})
}

// RecentOrders returns the most recent 5 orders for the dashboard.
func (h *AdminHandler) RecentOrders(c *fiber.Ctx) error {
	var orders []models.Order
	if err := h.db.Preload("Items").Preload("User").
		Order("placed_at desc").
		Limit(5).
		Find(&orders).Error; err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    orders,
	})
}
