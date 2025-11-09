package handlers

import (
	"AI_DETECTOR/go-backend/internal/database"
	"AI_DETECTOR/go-backend/internal/models"
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var userSessions = make(map[string]int)

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func generateSessionID(email string) string {
	return email + "-" + time.Now().Format("20060102150405") + "-" + time.Now().Format("000000000")
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func validateEmail(email string) bool {
	return emailRegex.MatchString(email) && len(email) <= 255
}

func validatePassword(password string) bool {
	if len(password) < 8 || len(password) > 72 {
		return false
	}
	hasLetter := false
	hasNumber := false
	for _, char := range password {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') {
			hasLetter = true
		}
		if char >= '0' && char <= '9' {
			hasNumber = true
		}
	}
	return hasLetter && hasNumber
}

func validateUsername(username string) bool {
	if len(username) < 3 || len(username) > 30 {
		return false
	}
	usernameRegex := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	return usernameRegex.MatchString(username)
}

func getUserIDFromCookie(r *http.Request) (int, bool) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return 0, false
	}
	userID, exists := userSessions[cookie.Value]
	return userID, exists
}

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5000")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Cookie")
	w.Header().Set("Content-Type", "application/json")
}

func Register(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" || req.Username == "" {
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}

	if !validateEmail(req.Email) {
		http.Error(w, "Invalid email format", http.StatusBadRequest)
		return
	}

	if !validatePassword(req.Password) {
		http.Error(w, "Password must be 8-72 characters with at least one letter and one number", http.StatusBadRequest)
		return
	}

	if !validateUsername(req.Username) {
		http.Error(w, "Username must be 3-30 characters, alphanumeric and underscore only", http.StatusBadRequest)
		return
	}

	passwordHash, err := hashPassword(req.Password)
	if err != nil {
		log.Printf("Password hashing error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	result, err := database.DB.Exec(
		"INSERT INTO users (email, username,  password_hash) VALUES (?, ?, ?)",
		req.Email, req.Username, passwordHash,
	)
	if err != nil {
		log.Printf("Registration failed: %v", err)
		errMsg := err.Error()
		if strings.Contains(errMsg, "UNIQUE constraint failed: users.username") {
			http.Error(w, "Username already taken", http.StatusConflict)
		} else if strings.Contains(errMsg, "UNIQUE constraint failed: users.email") {
			http.Error(w, "Email already registered", http.StatusConflict)
		} else {
			http.Error(w, "User already exists", http.StatusConflict)
		}
		return
	}

	userID, _ := result.LastInsertId()

	user := models.User{
		ID:        int(userID),
		Email:     req.Email,
		Username:  req.Username,
		CreatedAt: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
	log.Printf("User registered: %s", req.Email)
}

func Login(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		http.Error(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	if !validateEmail(req.Email) {
		http.Error(w, "Invalid email format", http.StatusBadRequest)
		return
	}

	var user models.User
	var storedHash string
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	err := database.DB.QueryRowContext(ctx,
		"SELECT id, email, username, password_hash, created_at FROM users WHERE email = ?",
		req.Email,
	).Scan(&user.ID, &user.Email, &user.Username, &storedHash, &user.CreatedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Printf("Login error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(req.Password))
	if err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	for sessionKey, userID := range userSessions {
		if userID == user.ID {
			delete(userSessions, sessionKey)
		}
	}

	oldCookie, err := r.Cookie("session_id")
	if err == nil {
		delete(userSessions, oldCookie.Value)
		http.SetCookie(w, &http.Cookie{
			Name:     "session_id",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
	}

	sessionID := generateSessionID(req.Email)
	userSessions[sessionID] = user.ID

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
	log.Printf("User logged in: %s", req.Email)

}

func Logout(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	cookie, err := r.Cookie("session_id")
	if err == nil {
		delete(userSessions, cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Logged out"))
}

func GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, exists := getUserIDFromCookie(r)
	if !exists {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var user models.User
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	err := database.DB.QueryRowContext(ctx,
		"SELECT id, email, username, created_at FROM users WHERE id = ?",
		userID,
	).Scan(&user.ID, &user.Email, &user.Username, &user.CreatedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("GetCurrentUser error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func CreateSession(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, exists := getUserIDFromCookie(r)
	if !exists {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.CreateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	result, err := database.DB.Exec(
		"INSERT INTO sessions (user_id, notes) VALUES (?, ?)",
		userID, req.Notes,
	)
	if err != nil {
		log.Printf("CreateSession error: %v", err)
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	sessionID, _ := result.LastInsertId()

	session := models.Session{
		ID:        int(sessionID),
		UserID:    userID,
		StartTime: time.Now(),
		Status:    "active",
		Notes:     req.Notes,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(session)
	log.Printf("Session created: ID=%d for user %d", sessionID, userID)
}

func GetSessions(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, exists := getUserIDFromCookie(r)
	if !exists {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := database.DB.Query(
		"SELECT id, user_id, start_time, end_time, status, notes FROM sessions WHERE user_id = ? ORDER BY start_time DESC",
		userID,
	)

	if err != nil {
		http.Error(w, "Failed to fetch sessions", http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	var sessions []models.Session
	for rows.Next() {
		var s models.Session
		var endTime sql.NullTime
		err := rows.Scan(&s.ID, &s.UserID, &s.StartTime, &endTime, &s.Status, &s.Notes)
		if err != nil {
			continue
		}
		if endTime.Valid {
			s.EndTime = &endTime.Time
		}
		sessions = append(sessions, s)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

func EndSession(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, exists := getUserIDFromCookie(r)
	if !exists {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	sessionIDStr := r.URL.Query().Get("id")
	sessionID, err := strconv.Atoi(sessionIDStr)
	if err != nil {
		http.Error(w, "Invalid session ID", http.StatusBadRequest)
		return
	}

	result, err := database.DB.Exec(
		"UPDATE sessions SET end_time = ?, status = 'completed' WHERE id = ? AND user_id = ?",
		time.Now(), sessionID, userID,
	)
	if err != nil {
		log.Printf("Failed to end session: %v", err)
		http.Error(w, "Failed to end session", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Session not found or does not belong to user", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Session ended"))
	log.Printf("Session ended: %d", sessionID)
}

func DeleteSession(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method != http.MethodDelete && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, exists := getUserIDFromCookie(r)
	if !exists {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	sessionIDStr := r.URL.Query().Get("id")
	sessionID, err := strconv.Atoi(sessionIDStr)
	if err != nil {
		http.Error(w, "Invalid session ID", http.StatusBadRequest)
		return
	}

	// Сначала проверяем, что сеанс принадлежит пользователю
	var sessionUserID int
	err = database.DB.QueryRow(
		"SELECT user_id FROM sessions WHERE id = ?",
		sessionID,
	).Scan(&sessionUserID)
	if err == sql.ErrNoRows {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Failed to verify session: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if sessionUserID != userID {
		http.Error(w, "Unauthorized: session does not belong to user", http.StatusForbidden)
		return
	}

	// Удаляем события сеанса
	_, err = database.DB.Exec("DELETE FROM events WHERE session_id = ?", sessionID)
	if err != nil {
		log.Printf("Failed to delete events: %v", err)
		// Продолжаем удаление сеанса даже если не удалось удалить события
	}

	// Удаляем сеанс
	result, err := database.DB.Exec("DELETE FROM sessions WHERE id = ? AND user_id = ?", sessionID, userID)
	if err != nil {
		log.Printf("Failed to delete session: %v", err)
		http.Error(w, "Failed to delete session", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Session deleted"))
	log.Printf("Session deleted: %d", sessionID)
}

func SaveEvent(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, exists := getUserIDFromCookie(r)
	if !exists {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	var sessionUserID int
	err := database.DB.QueryRowContext(ctx,
		"SELECT user_id FROM sessions WHERE id = ?",
		req.SessionID,
	).Scan(&sessionUserID)
	if err == sql.ErrNoRows {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Failed to verify session: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if sessionUserID != userID {
		http.Error(w, "Unauthorized: session does not belong to user", http.StatusForbidden)
		return
	}

	isDrowsyInt := 0
	if req.IsDrowsy {
		isDrowsyInt = 1
	}

	result, err := database.DB.Exec(
		"INSERT INTO events (session_id, drowsiness_score, is_drowsy) VALUES (?, ?, ?)",
		req.SessionID, req.DrowsinessScore, isDrowsyInt,
	)

	if err != nil {
		log.Printf("Failed to save event: %v", err)
		http.Error(w, "Failed to save event", http.StatusInternalServerError)
		return
	}

	eventID, _ := result.LastInsertId()

	event := models.Event{
		ID:              int(eventID),
		SessionID:       req.SessionID,
		DrowsinessScore: req.DrowsinessScore,
		IsDrowsy:        req.IsDrowsy,
		Timestamp:       time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(event)
}

func GetEvents(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, exists := getUserIDFromCookie(r)
	if !exists {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	sessionIDStr := r.URL.Query().Get("session_id")
	sessionID, err := strconv.Atoi(sessionIDStr)
	if err != nil {
		http.Error(w, "Invalid session ID", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	var sessionUserID int
	err = database.DB.QueryRowContext(ctx,
		"SELECT user_id FROM sessions WHERE id = ?",
		sessionID,
	).Scan(&sessionUserID)
	if err == sql.ErrNoRows {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Failed to verify session: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if sessionUserID != userID {
		http.Error(w, "Unauthorized: session does not belong to user", http.StatusForbidden)
		return
	}

	rows, err := database.DB.Query(
		"SELECT id, session_id, drowsiness_score, is_drowsy, timestamp FROM events WHERE session_id = ? ORDER BY timestamp DESC",
		sessionID,
	)

	if err != nil {
		http.Error(w, "Failed to fetch events", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		var e models.Event
		var isDrowsyInt int
		err := rows.Scan(&e.ID, &e.SessionID, &e.DrowsinessScore, &isDrowsyInt, &e.Timestamp)
		if err != nil {
			continue
		}
		e.IsDrowsy = isDrowsyInt == 1
		events = append(events, e)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}
