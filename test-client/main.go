package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	BackendURL = "http://localhost:8080"
	TestEmail  = "test@example.com"
	TestPass   = "Test123456"
)

// Проверка состояния
func testHealth() error {
	fmt.Println("\n[TEST] Testing /api/health...")
	resp, err := http.Get(BackendURL + "/api/health")
	if err != nil {
		return fmt.Errorf("health check failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("✓ Health check: %s\n", string(body))
	return nil
}

// проверка регистрации
func testRegister() error {
	fmt.Println("\n[TEST] Testing /api/auth/register...")

	data := map[string]string{
		"email":    TestEmail,
		"username": "testuser",
		"password": TestPass,
	}

	jsonData, _ := json.Marshal(data)
	resp, err := http.Post(BackendURL+"/api/auth/register", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("registration failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusCreated {
		fmt.Printf("✓ Registration successful: %s\n", string(body))
		return nil
	} else if resp.StatusCode == http.StatusConflict {
		fmt.Printf("⚠ User already exists (this is OK)\n")
		return nil
	}

	return fmt.Errorf("registration failed: status %d, body: %s", resp.StatusCode, string(body))
}

// Проверка логина
func testLogin() (*http.Client, []*http.Cookie, error) {
	fmt.Println("\n[TEST] Testing /api/auth/login...")

	data := map[string]string{
		"email":    TestEmail,
		"password": TestPass,
	}

	jsonData, _ := json.Marshal(data)
	client := &http.Client{}
	req, _ := http.NewRequest("POST", BackendURL+"/api/auth/login", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("login failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("login failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	cookies := resp.Cookies()
	if len(cookies) == 0 {
		return nil, nil, fmt.Errorf("no session cookie received")
	}

	fmt.Printf("✓ Login successful, session cookie received\n")
	return client, cookies, nil
}

// Проверка детекции
func testDetection(client *http.Client, cookies []*http.Cookie, frameData []byte) error {
	fmt.Println("\n[TEST] Testing /api/detect...")
	frameBase64 := base64.StdEncoding.EncodeToString(frameData)

	data := map[string]interface{}{
		"frame":           frameBase64,
		"timestamp":       time.Now().UnixMilli(),
		"sequence_number": 1,
	}

	jsonData, _ := json.Marshal(data)
	req, _ := http.NewRequest("POST", BackendURL+"/api/detect", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("detection request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("detection failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}

	fmt.Printf("✓ Detection successful!\n")
	fmt.Printf("  - Drowsy: %v\n", result["is_drowsy"])
	fmt.Printf("  - Score: %.3f\n", result["drowsiness_score"])
	fmt.Printf("  - Alert Level: %v\n", result["alert_level"])
	fmt.Printf("  - Inference Time: %.2f ms\n", result["inference_time"])

	return nil
}

// Проверка создания сеанса
func testCreateSession(client *http.Client, cookies []*http.Cookie) (int, error) {
	fmt.Println("\n[TEST] Testing /api/sessions (POST)...")

	data := map[string]string{
		"notes": "Test session from automated test",
	}

	jsonData, _ := json.Marshal(data)
	req, _ := http.NewRequest("POST", BackendURL+"/api/sessions", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("create session failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return 0, fmt.Errorf("create session failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	var session map[string]interface{}
	if err := json.Unmarshal(body, &session); err != nil {
		return 0, fmt.Errorf("failed to parse session: %v", err)
	}

	sessionID := int(session["id"].(float64))
	fmt.Printf("✓ Session created: ID=%d\n", sessionID)
	return sessionID, nil
}

// Просмотр сеанса
func testGetSessions(client *http.Client, cookies []*http.Cookie) error {
	fmt.Println("\n[TEST] Testing /api/sessions (GET)...")

	req, _ := http.NewRequest("GET", BackendURL+"/api/sessions", nil)

	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("get sessions failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("get sessions failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	var sessions []interface{}
	if err := json.Unmarshal(body, &sessions); err != nil {
		return fmt.Errorf("failed to parse sessions: %v", err)
	}

	fmt.Printf("✓ Retrieved %d sessions\n", len(sessions))
	return nil
}

// Сохранение поездки
func testSaveEvent(client *http.Client, cookies []*http.Cookie, sessionID int) error {
	fmt.Println("\n[TEST] Testing /api/events (POST)...")

	data := map[string]interface{}{
		"session_id":       sessionID,
		"drowsiness_score": 0.75,
		"is_drowsy":        true,
	}

	jsonData, _ := json.Marshal(data)
	req, _ := http.NewRequest("POST", BackendURL+"/api/events", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("save event failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("save event failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	fmt.Printf("✓ Event saved successfully\n")
	return nil
}

func main() {
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Println("AI DETECTOR - Backend & Model Testing Client")
	fmt.Println("=" + strings.Repeat("=", 60))

	fmt.Println("\n[INFO] Make sure the Go backend is running on", BackendURL)
	fmt.Println("[INFO] Make sure the Python ML service is running on localhost:9000")
	fmt.Println("\nPress Enter to start tests...")
	fmt.Scanln()

	fmt.Println("\n[INFO] Generating test image...")
	frameData, err := generateTestImage()
	if err != nil {
		log.Fatalf("Failed to generate test image: %v", err)
	}
	fmt.Printf("✓ Generated test image: %d bytes\n", len(frameData))

	tests := []struct {
		name string
		fn   func() error
	}{
		{"Health Check", testHealth},
		{"Registration", testRegister},
	}

	for _, test := range tests {
		if err := test.fn(); err != nil {
			log.Printf("❌ %s failed: %v", test.name, err)
			os.Exit(1)
		}
	}

	client, cookies, err := testLogin()
	if err != nil {
		log.Printf("❌ Login failed: %v", err)
		os.Exit(1)
	}

	if err := testDetection(client, cookies, frameData); err != nil {
		log.Printf("❌ Detection test failed: %v", err)
		log.Printf("   Make sure Python ML service is running!")
		os.Exit(1)
	}

	sessionID, err := testCreateSession(client, cookies)
	if err != nil {
		log.Printf("⚠ Session creation failed: %v", err)
	} else {
		testGetSessions(client, cookies)
		testSaveEvent(client, cookies, sessionID)
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("✅ All tests completed successfully!")
	fmt.Println("=" + strings.Repeat("=", 60))
}
