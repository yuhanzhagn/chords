package app

import (
	"backend/internal/model"
	"log"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func InitDB(db *gorm.DB) error {
	if err := CheckMessageSchema(db); err != nil {
		return err
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.ChatRoom{},
		&model.UserSession{},
		&model.UserChatRoom{},
	); err != nil {
		return err
	}
	return MigrateMessageSchema(db)
}

func ConnectDB(cfg *Config) (*gorm.DB, error) {
	/* cfg, err := config.LoadConfig("config.yaml")
	   if err != nil {
	       log.Fatal("Failed to load config:", err)
	   } */

	// Only handling sqlite for now
	if cfg.Database.Dialect != "sqlite" {
		log.Fatal("Unsupported DB dialect:", cfg.Database.Dialect)
	}
	db, err := gorm.Open(sqlite.Open(cfg.Database.DSN), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	log.Println("Database connected.")
	return db, err
}
