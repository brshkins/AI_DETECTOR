package database

import (
	"AI_DETECTOR/go-backend/internal/config"
	"database/sql"
	"github.com/pressly/goose/v3"
	"log"
	"path/filepath"
	"time"

	_ "github.com/jackc/pgx/v4/stdlib"
)

var DB *sql.DB

func InitDB(cfg *config.Config) error {
	var err error
	log.Printf("Connecting to database: %s", cfg.DSNForLog())
	DB, err = sql.Open("pgx", cfg.DSN())
	if err != nil {
		return err
	}

	// проверка соединения
	if err = DB.Ping(); err != nil {
		return err
	}

	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(5)
	DB.SetConnMaxLifetime(5 * time.Minute)

	// создание таблиц
	if err = runMigrations(); err != nil {
		return err
	}

	log.Println("PostgreSQL database initialized")
	return nil
}

func runMigrations() error {
	// драйвер бд
	log.Println("Running database migrations...")

	// применение миграций
	migrationDir := filepath.Join(".", "migrations")
	if err := goose.Up(DB, migrationDir); err != nil {
		return err
	}
	log.Println("Database migrations successfully applied")
	return nil
}

func CloseDB() {
	if DB != nil {
		DB.Close()
		log.Println("DB closed")
	}
}
