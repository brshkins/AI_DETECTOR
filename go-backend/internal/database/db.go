package database

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB
var currentDBPath string

func InitDB(path string) error {
	var err error
	currentDBPath = path
	DB, err = sql.Open("sqlite3", path)
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
	if err = createTables(); err != nil {
		return err
	}

	log.Println("SQLite database initialized")
	return nil
}

func createTables() error {
	schemaPaths := []string{
		"schema.sql",
		"go-backend/schema.sql",
		"./go-backend/schema.sql",
	}

	if currentDBPath != "" {
		schemaPaths = append(schemaPaths, filepath.Join(filepath.Dir(currentDBPath), "schema.sql"))
	}

	var schema []byte
	var err error
	for _, path := range schemaPaths {
		schema, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}

	if err != nil {
		// If schema.sql not found, use manual creation
		return createTablesManually()
	}

	_, err = DB.Exec(string(schema))
	return err
}

func createTablesManually() error {
	schema := `
		CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT UNIQUE NOT NULL,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		start_time DATETIME DEFAULT CURRENT_TIMESTAMP,
		end_time DATETIME,
		status TEXT DEFAULT 'active',
		notes TEXT,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);

	CREATE TABLE IF NOT EXISTS events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id INTEGER NOT NULL,
		drowsiness_score REAL NOT NULL,
		is_drowsy INTEGER DEFAULT 0,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (session_id) REFERENCES sessions(id)
	);

	CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
	CREATE INDEX IF NOT EXISTS idx_events_session ON events(session_id);
	`

	_, err := DB.Exec(schema)
	return err
}

func CloseDB() {
	if DB != nil {
		DB.Close()
		log.Println("DB closed")
	}
}
