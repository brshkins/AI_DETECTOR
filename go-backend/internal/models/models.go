package models

import "time"

type User struct {
	ID           int       `json:"id"`
	Email        string    `json:"email"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Session struct {
	ID        int        `json:"id"`
	UserID    int        `json:"user_id"`
	StartTime time.Time  `json:"start_time"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	Status    string     `json:"status"`
	Notes     string     `json:"notes,omitempty"`
}

type Event struct {
	ID              int       `json:"id"`
	SessionID       int       `json:"session_id"`
	DrowsinessScore float64   `json:"drowsiness_score"`
	IsDrowsy        bool      `json:"is_drowsy"`
	Timestamp       time.Time `json:"timestamp"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type CreateSessionRequest struct {
	Notes string `json:"notes,omitempty"`
}

type CreateEventRequest struct {
	SessionID       int     `json:"session_id"`
	DrowsinessScore float64 `json:"drowsiness_score"`
	IsDrowsy        bool    `json:"is_drowsy"`
}
