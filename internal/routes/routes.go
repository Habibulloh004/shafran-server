package routes

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"github.com/example/shafran/internal/config"
	"github.com/example/shafran/internal/handlers"
	"github.com/example/shafran/internal/middleware"
	"github.com/example/shafran/internal/services"
)

// Register wires up all HTTP routes.
func Register(app *fiber.App, db *gorm.DB, cfg *config.Config) {
	// Initialize Telegram service
	telegramService := services.NewTelegramService(cfg.TelegramBotToken, cfg.TelegramAdminChat)

	authHandler := handlers.NewAuthHandler(db, cfg)
	catalogHandler := handlers.NewCatalogHandler(db)
	productHandler := handlers.NewProductHandler(db)
	orderHandler := handlers.NewOrderHandler(db, telegramService)
	paymeHandler := handlers.NewPaymeHandler(db, cfg.PaymeMerchantID, telegramService)
	profileHandler := handlers.NewProfileHandler(db)
	marketingHandler := handlers.NewMarketingHandler(db)
	billzHandler := handlers.NewBillzHandler()

	api := app.Group("/api")

	// Auth routes
	auth := api.Group("/auth")
	auth.Post("/register", authHandler.Register)
	auth.Post("/login", authHandler.Login)
	auth.Post("/verify", authHandler.Verify)

	// Catalog routes
	categories := api.Group("/categories")
	categories.Get("/", catalogHandler.ListCategories)
	categories.Post("/", catalogHandler.CreateCategory)
	categories.Get("/:id", catalogHandler.GetCategory)
	categories.Put("/:id", catalogHandler.UpdateCategory)
	categories.Delete("/:id", catalogHandler.DeleteCategory)

	brands := api.Group("/brands")
	brands.Get("/", catalogHandler.ListBrands)
	brands.Post("/", catalogHandler.CreateBrand)
	brands.Get("/:id", catalogHandler.GetBrand)
	brands.Put("/:id", catalogHandler.UpdateBrand)
	brands.Delete("/:id", catalogHandler.DeleteBrand)

	fragranceNotes := api.Group("/fragrance-notes")
	fragranceNotes.Get("/", catalogHandler.ListFragranceNotes)
	fragranceNotes.Post("/", catalogHandler.CreateFragranceNote)
	fragranceNotes.Get("/:id", catalogHandler.GetFragranceNote)
	fragranceNotes.Put("/:id", catalogHandler.UpdateFragranceNote)
	fragranceNotes.Delete("/:id", catalogHandler.DeleteFragranceNote)

	seasons := api.Group("/seasons")
	seasons.Get("/", catalogHandler.ListSeasons)
	seasons.Post("/", catalogHandler.CreateSeason)
	seasons.Get("/:id", catalogHandler.GetSeason)
	seasons.Put("/:id", catalogHandler.UpdateSeason)
	seasons.Delete("/:id", catalogHandler.DeleteSeason)

	productTypes := api.Group("/product-types")
	productTypes.Get("/", catalogHandler.ListProductTypes)
	productTypes.Post("/", catalogHandler.CreateProductType)
	productTypes.Get("/:id", catalogHandler.GetProductType)
	productTypes.Put("/:id", catalogHandler.UpdateProductType)
	productTypes.Delete("/:id", catalogHandler.DeleteProductType)

	// Products
	products := api.Group("/products")
	productHandler.RegisterProductRoutes(products)

	// Marketing resources
	api.Get("/banner", marketingHandler.ListBanners)
	api.Post("/banner", marketingHandler.CreateBanner)
	api.Put("/banner/:id", marketingHandler.UpdateBanner)
	api.Delete("/banner/:id", marketingHandler.DeleteBanner)

	billz := api.Group("/billz")
	billz.All("/", billzHandler.Proxy)
	billz.All("/*", billzHandler.Proxy)

	pickup := api.Group("/pickup-branches")
	pickup.Get("/", marketingHandler.ListPickupBranches)
	pickup.Post("/", marketingHandler.CreatePickupBranch)
	pickup.Put("/:id", marketingHandler.UpdatePickupBranch)
	pickup.Delete("/:id", marketingHandler.DeletePickupBranch)

	payments := api.Group("/payment-providers")
	payments.Get("/", marketingHandler.ListPaymentProviders)
	payments.Post("/", marketingHandler.CreatePaymentProvider)
	payments.Put("/:id", marketingHandler.UpdatePaymentProvider)
	payments.Delete("/:id", marketingHandler.DeletePaymentProvider)

	// Payme payment routes
	payme := api.Group("/payme")
	payme.Get("/transactions", paymeHandler.ListTransactions)
	payme.Post("/checkout", paymeHandler.Checkout)
	payme.Post("/pay", middleware.PaymeAuthMiddleware(cfg.PaymeMerchantKey), paymeHandler.Pay)
	payme.Post("/fake-transaction", paymeHandler.CreateFakeTransaction)

	// Protected routes
	protected := api.Group("", middleware.AuthMiddleware(cfg))

	protected.Post("/orders", orderHandler.CreateOrder)
	protected.Get("/orders", orderHandler.ListOrders)
	protected.Get("/orders/:id", orderHandler.GetOrder)

	protected.Get("/profile", profileHandler.GetProfile)
	protected.Put("/profile", profileHandler.UpdateProfile)
	protected.Get("/profile/addresses", profileHandler.ListAddresses)
	protected.Post("/profile/addresses", profileHandler.CreateAddress)
	protected.Put("/profile/addresses/:id", profileHandler.UpdateAddress)
	protected.Delete("/profile/addresses/:id", profileHandler.DeleteAddress)
	protected.Get("/profile/bonus", profileHandler.ListBonusTransactions)
}
