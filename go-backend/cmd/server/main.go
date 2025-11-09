package main

import (
	"AI_DETECTOR/go-backend/internal/config"
	"AI_DETECTOR/go-backend/internal/database"
	"AI_DETECTOR/go-backend/internal/handlers"
	"AI_DETECTOR/go-backend/internal/models"
	"AI_DETECTOR/go-backend/internal/services"
	"AI_DETECTOR/go-backend/pkg/pb"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var (
	grpcServer      *grpc.Server
	httpServer      *http.Server
	grpcClient      *services.GRPCClient
	appConfig       *config.Config
	serverStartTime time.Time

	wsClients = &WebSocketClients{
		clients: make(map[string]*WebSocketClient),
	}
)

type WebSocketClient struct {
	conn     *websocket.Conn
	clientID string
	send     chan interface{}
	mu       sync.Mutex
}

type WebSocketClients struct {
	mu      sync.RWMutex
	clients map[string]*WebSocketClient
	count   int32
}

type WebSocketMessage struct {
	Type      string      `json:"type"`
	Payload   interface{} `json:"payload"`
	ClientID  string      `json:"client_id,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

func enableCORS(w http.ResponseWriter, cfg *config.Config) {
	origin := cfg.CORSOrigins
	if origin == "" {
		origin = "http://localhost:5000"
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Cookie")
}

func main() {
	httpPort := flag.String("http-port", ":8080", "HTTP port")
	grpcPort := flag.String("grpc-port", ":50051", "gRPC port")
	pythonURL := flag.String("python-url", "localhost:9000", "Python service URL")
	flag.Parse()

	cfg := config.LoadConfig()
	appConfig = cfg
	serverStartTime = time.Now()

	log.Println("Starting...")
	log.Printf("gRPC port: %s", *grpcPort)
	log.Printf("HTTP port: %s", *httpPort)
	log.Printf("Python service: %s", *pythonURL)
	log.Printf("Environment: %s", cfg.Environment)

	log.Println("Initializing database...")
	if err := database.InitDB("./app.db"); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.CloseDB()
	log.Println("Database initialized successfully")

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Подключение к Python
	var err error
	grpcClient, err = services.NewGRPCClient(*pythonURL)
	if err != nil {
		log.Printf("Python service unavailable: %v", err)
		log.Println("Continuing without Python (for testing)")
		grpcClient = nil
	} else {
		log.Printf("gRPC client created, verifying connection...")
		if !grpcClient.HealthCheck() {
			log.Printf("WARNING: Python service connected but health check failed - connection may be unstable")
		} else {
			log.Printf("Python service connected and healthy")
		}
		defer func() {
			if grpcClient != nil {
				log.Println("Closing gRPC client connection...")
				grpcClient.Close()
			}
		}()
	}

	// gRPC сервер
	grpcServer = grpc.NewServer(
		grpc.MaxRecvMsgSize(50*1024*1024),
		grpc.MaxSendMsgSize(50*1024*1024),
	)
	grpcHandler := handlers.NewGRPCHandler(grpcClient)
	pb.RegisterDrowsinessDetectionServer(grpcServer, grpcHandler)

	log.Println("\n Starting gRPC server...")
	go startGRPCServer(*grpcPort)

	log.Println("\n Starting HTTP server...")
	go startHTTPServer(*httpPort)

	// Ждём сигнала
	<-done
	log.Println("Shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stopped := make(chan struct{})
	go func() {
		log.Println("Stopping gRPC server...")
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		log.Println("Stopped")
	case <-shutdownCtx.Done():
		log.Println("Forced shutdown")
		grpcServer.Stop()
	}

	if httpServer != nil {
		httpShutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		log.Println("Stopping HTTP server...")
		if err := httpServer.Shutdown(httpShutdownCtx); err != nil {
			log.Printf("Error shutting down HTTP server: %v", err)
		} else {
			log.Println("HTTP server gracefully stopped")
		}
	}
	log.Println("Closing WebSocket connections...")
	closeAllWebSocketConnections()
	log.Println("All WebSocket connections closed...")

	log.Println("Goodbye!")
}

func startGRPCServer(grpcPort string) {
	port := grpcPort
	if len(port) > 0 && port[0] == ':' {
		port = port[1:]
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("failed to listen on gRPC port %v", err)
	}

	log.Printf("gRPC server listening on port %s", port)
	log.Println("Waiting for gRPC connections")

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve gRPC server %v", err)
	}
}

func startHTTPServer(httpPort string) {
	port := httpPort
	if len(port) > 0 && port[0] == ':' {
		port = port[1:]
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/ws", handleWebSocket)

	mux.HandleFunc("/api/detect", handleDetect)
	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/api/metrics", handleMetrics)

	mux.HandleFunc("/api/auth/register", handlers.Register)
	mux.HandleFunc("/api/auth/login", handlers.Login)
	mux.HandleFunc("/api/auth/logout", handlers.Logout)

	mux.HandleFunc("/api/sessions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handlers.GetSessions(w, r)
		} else if r.Method == http.MethodPost {
			handlers.CreateSession(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/sessions/end", handlers.EndSession)

	mux.HandleFunc("/api/events", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handlers.GetEvents(w, r)
		} else if r.Method == http.MethodPost {
			handlers.SaveEvent(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	log.Println("Database endpoints registered")

	httpServer = &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("HTTP server listening on port %s", port)
	log.Printf("WebSocket:  ws://localhost:%s/ws", port)
	log.Printf("REST API:   http://localhost:%s/api/*", port)

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to serve HTTP: %v", err)
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			if appConfig == nil {
				return false
			}
			origin := r.Header.Get("Origin")
			allowedOrigins := appConfig.CORSOrigins
			if allowedOrigins == "" {
				allowedOrigins = "http://localhost:5000"
			}
			if appConfig.IsDev() {
				return origin == allowedOrigins || origin == "" ||
					(origin[:7] == "http://" && (origin[7:17] == "localhost" || origin[7:9] == "127"))
			}
			return origin == allowedOrigins
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}

	clientID := r.URL.Query().Get("clientId")
	if clientID == "" {
		clientID = generateClientID()
	}

	log.Printf("WebSocket client connected: %s", clientID)

	// Структура клиента
	client := &WebSocketClient{
		conn:     conn,
		clientID: clientID,
		send:     make(chan interface{}, 256),
	}

	// Регистрируем клиента
	wsClients.mu.Lock()
	wsClients.clients[clientID] = client
	wsClients.mu.Unlock()
	atomic.AddInt32(&wsClients.count, 1)

	defer func() {
		// Удаляем клиента при отключении
		wsClients.mu.Lock()
		delete(wsClients.clients, clientID)
		wsClients.mu.Unlock()
		atomic.AddInt32(&wsClients.count, -1)

		conn.Close()
		log.Printf("WebSocket client disconnected: %s", clientID)
	}()

	// Запускаем цикл чтения и записи
	go readPump(client)
	go writePump(client)

	// Отправляем приветственное сообщение
	welcomeMsg := WebSocketMessage{
		Type:      "WELCOME",
		ClientID:  clientID,
		Timestamp: time.Now().Unix(),
		Payload: map[string]interface{}{
			"message": "Connected to Drowsiness Detection Server",
			"version": "1.0",
		},
	}

	client.send <- welcomeMsg
}

// Цикл чтения из WebSocket
func readPump(client *WebSocketClient) {
	defer func() {
		client.conn.Close()
	}()

	client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var msg WebSocketMessage
		err := client.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error for %s: %v", client.clientID, err)
			}
			break
		}

		log.Printf("Received from %s: %s", client.clientID, msg.Type)

		// Обработка сообщений
		switch msg.Type {
		case "PING":
			client.send <- WebSocketMessage{
				Type:      "PONG",
				ClientID:  client.clientID,
				Timestamp: time.Now().Unix(),
			}

		case "FRAME":
			// Обработка видео кадра
			response := WebSocketMessage{
				Type:      "FRAME_RECEIVED",
				ClientID:  client.clientID,
				Timestamp: time.Now().Unix(),
				Payload: map[string]interface{}{
					"status": "processed",
				},
			}
			client.send <- response

		default:
			log.Printf("Unknown message type: %s", msg.Type)
		}
	}
}

// Цикл отправки в WebSocket
func writePump(client *WebSocketClient) {
	ticker := time.NewTicker(5 * time.Second)
	defer func() {
		ticker.Stop()
		client.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

			if !ok {
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.conn.WriteJSON(msg); err != nil {
				return
			}

		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Обработчик REST API - Обнаружение
func handleDetect(w http.ResponseWriter, r *http.Request) {
	enableCORS(w, appConfig)

	// Обработка preflight OPTIONS запросов
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent) // 204, не 200!
		return
	}

	// Проверка метода
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Method not allowed",
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024)

	// Парсирование запроса
	var req models.DetectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Проверка пустого кадра
	if req.Frame == "" {
		http.Error(w, "Empty frame data", http.StatusBadRequest)
		return
	}

	if len(req.Frame) > 8*1024*1024 { //
		http.Error(w, "Frame data too large (max 8MB)", http.StatusBadRequest)
		return
	}

	// Декодирование base64 в байты
	frameBytes, err := base64.StdEncoding.DecodeString(req.Frame)
	if err != nil {
		log.Printf("Base64 decode error: %v", err)
		http.Error(w, "Invalid frame data format", http.StatusBadRequest)
		return
	}

	log.Printf("Received frame: size=%d bytes, sequence=%d", len(frameBytes), req.SequenceNumber)

	// Проверка доступности gRPC клиента
	if grpcClient == nil {
		log.Println("WARNING: gRPC client is nil - ML service unavailable")
		log.Println("DEBUG: This means the connection failed at startup or was never established")
		http.Error(w, "ML service unavailable", http.StatusServiceUnavailable)
		return
	}

	if !grpcClient.HealthCheck() {
		log.Printf("WARNING: gRPC client exists but connection is not healthy")
		http.Error(w, "ML service connection lost", http.StatusServiceUnavailable)
		return
	}

	// Создание protobuf сообщения для ML сервиса
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	videoFrame := &pb.VideoFrame{
		FrameData:      frameBytes,
		Timestamp:      req.Timestamp,
		SequenceNumber: req.SequenceNumber,
	}

	// Отправка на Python ML сервис через gRPC
	result, err := grpcClient.ProcessFrame(ctx, videoFrame)
	if err != nil {
		log.Printf("ML processing error: %v", err)
		http.Error(w, "Processing failed", http.StatusInternalServerError)
		return
	}

	// Успешный ответ
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"is_drowsy":        result.IsDrowsy,
		"drowsiness_score": float64(result.DrowsinessScore),
		"alert_level":      result.AlertLevel,
		"timestamp":        time.Now().Unix(),
		"inference_time":   result.InferenceTimeMs,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Encode response error: %v", err)
	}

	log.Printf("Detection result: drowsy=%v, score=%.2f", result.IsDrowsy, result.DrowsinessScore)
}

// Обработчик REST API - Проверка здоровья
func handleHealth(w http.ResponseWriter, r *http.Request) {
	enableCORS(w, appConfig)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Method not allowed",
		})
		return
	}

	grpcOk := false
	if grpcClient != nil {
		grpcOk = grpcClient.HealthCheck()
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"grpc_ok":   grpcOk,
		"http_ok":   true,
		"timestamp": time.Now().Unix(),
	})
}

// Обработчик REST API - Метрики
func handleMetrics(w http.ResponseWriter, r *http.Request) {
	enableCORS(w, appConfig)
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Method not allowed",
		})
		return
	}

	log.Println("/api/metrics - Metrics request")

	wsClients.mu.RLock()
	activeClients := len(wsClients.clients)
	wsClients.mu.RUnlock()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_frames":      0,
		"total_errors":      0,
		"active_clients":    activeClients,
		"avg_latency_ms":    0,
		"drowsy_detections": 0,
		"detection_rate":    0.0,
		"system_uptime_sec": int(time.Since(serverStartTime).Seconds()),
		"timestamp":         time.Now().Format(time.RFC3339),
	})
}

// Генерация ID клиента
func generateClientID() string {
	return "client-" + time.Now().Format("20060102150405")
}

func closeAllWebSocketConnections() {
	wsClients.mu.Lock()
	defer wsClients.mu.Unlock()

	for clientID, client := range wsClients.clients {
		close(client.send)
		client.conn.Close()
		log.Printf("Closed connection for client: %s", clientID)
	}
	wsClients.clients = make(map[string]*WebSocketClient)
}
