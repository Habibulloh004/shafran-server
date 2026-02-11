package database

import (
	"database/sql"
	"log"
	"net/url"
	"strings"

	"github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/example/shafran/internal/models"
)

var db *gorm.DB

// Connect initializes the database connection and runs migrations.
func Connect(dsn string) *gorm.DB {
	if db != nil {
		return db
	}

	if err := ensureDatabase(dsn); err != nil {
		log.Fatalf("failed to ensure database: %v", err)
	}

	conn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	if err := conn.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`).Error; err != nil {
		log.Printf("warning: failed to ensure uuid-ossp extension: %v", err)
	}

	if err := migrate(conn); err != nil {
		log.Fatalf("database migration failed: %v", err)
	}

	db = conn
	return db
}

// DB exposes the initialized gorm.DB instance.
func DB() *gorm.DB {
	return db
}

func migrate(conn *gorm.DB) error {
	migrations := []interface{}{
		&models.User{},
		&models.SMSVerification{},
		&models.Category{},
		&models.Brand{},
		&models.FragranceNote{},
		&models.Season{},
		&models.ProductType{},
		&models.Product{},
		&models.ProductVariant{},
		&models.ProductMedia{},
		&models.ProductSpecification{},
		&models.ProductDescriptionBlock{},
		&models.ProductHighlight{},
		&models.ProductRelation{},
		&models.Banner{},
		&models.PickupBranch{},
		&models.PaymentProvider{},
		&models.UserAddress{},
		&models.BonusTransaction{},
		&models.Order{},
		&models.OrderItem{},
		&models.PaymeTransaction{},
		&models.PasswordResetToken{},
		&models.FooterSettings{},
	}

	for _, migration := range migrations {
		if err := conn.AutoMigrate(migration); err != nil {
			return err
		}
	}

	return nil
}

func ensureDatabase(dsn string) error {
	if !strings.HasPrefix(dsn, "postgres://") && !strings.HasPrefix(dsn, "postgresql://") {
		return nil
	}

	parsed, err := url.Parse(dsn)
	if err != nil {
		return err
	}

	dbName := strings.TrimPrefix(parsed.Path, "/")
	if dbName == "" {
		return nil
	}

	parsed.Path = "/postgres"
	masterDSN := parsed.String()

	sqlDB, err := sql.Open("postgres", masterDSN)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	if err := sqlDB.Ping(); err != nil {
		return err
	}

	var exists bool
	if err := sqlDB.QueryRow("SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", dbName).Scan(&exists); err != nil {
		return err
	}

	if exists {
		return nil
	}

	_, err = sqlDB.Exec("CREATE DATABASE " + pq.QuoteIdentifier(dbName))
	return err
}
