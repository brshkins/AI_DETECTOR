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
	"net/url"
	"os"
	"os/signal"
	"strings"
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
	closed   int32 // Атомарный флаг для отслеживания закрытия
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

func enableCORS(w http.ResponseWriter, r *http.Request, cfg *config.Config) {
	origin := determineAllowOrigin(r.Header.Get("Origin"))
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Cookie")
}

func getAllowedOrigins() []string {
	if appConfig == nil || strings.TrimSpace(appConfig.CORSOrigins) == "" {
		return []string{"http://localhost:5000"}
	}

	raw := strings.Split(appConfig.CORSOrigins, ",")
	origins := make([]string, 0, len(raw))
	for _, origin := range raw {
		o := strings.TrimSpace(origin)
		if o == "" {
			continue
		}
		origins = append(origins, o)
	}

	if len(origins) == 0 {
		return []string{"http://localhost:5000"}
	}

	return origins
}

func isLocalOrigin(origin string) bool {
	if origin == "" {
		return false
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := u.Hostname()
	return host == "localhost" || host == "127.0.0.1"
}

func isOriginAllowed(origin string) bool {
	if appConfig == nil {
		return false
	}

	if origin == "" {
		return appConfig.IsDev()
	}

	for _, allowed := range getAllowedOrigins() {
		if allowed == "*" {
			return true
		}

		if origin == allowed {
			return true
		}

		if strings.Contains(allowed, "localhost") || strings.Contains(allowed, "127.0.0.1") {
			if isLocalOrigin(origin) {
				return true
			}
		}
	}

	if appConfig.IsDev() && isLocalOrigin(origin) {
		return true
	}

	return false
}

func determineAllowOrigin(requestOrigin string) string {
	if isOriginAllowed(requestOrigin) {
		return requestOrigin
	}

	origins := getAllowedOrigins()
	if len(origins) == 0 {
		return "*"
	}

	if origins[0] == "*" {
		return "*"
	}

	return origins[0]
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
	if err := database.InitDB(cfg); err != nil {
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

	// mux.HandleFunc("/api/detect", handleDetect)
	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/api/metrics", handleMetrics)

	mux.HandleFunc("/api/auth/register", handlers.Register)
	mux.HandleFunc("/api/auth/login", handlers.Login)
	mux.HandleFunc("/api/auth/logout", handlers.Logout)
	mux.HandleFunc("/api/auth/me", handlers.GetCurrentUser)

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
	mux.HandleFunc("/api/sessions/delete", handlers.DeleteSession)

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
	log.Printf("WebSocket connection attempt from %s, Origin: %s", r.RemoteAddr, r.Header.Get("Origin"))

	userID, exists := handlers.GetUserIDFromCookie(r)
	if !exists {
		log.Printf("WebSocket connection rejected: user not authenticated (no session_id cookie)")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	log.Printf("WebSocket connection authenticated for user ID: %d", userID)

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			allowed := isOriginAllowed(origin)
			log.Printf("WebSocket Origin check: %s -> %v", origin, allowed)
			return allowed
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}
	log.Printf("WebSocket upgrade successful")

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

	// Отправляем приветственное сообщение через горутину с задержкой
	// чтобы убедиться, что writePump запустился
	welcomeMsg := WebSocketMessage{
		Type:      "WELCOME",
		ClientID:  clientID,
		Timestamp: time.Now().Unix(),
		Payload: map[string]interface{}{
			"message": "Connected to Drowsiness Detection Server",
			"version": "1.0",
		},
	}

	go func() {
		time.Sleep(200 * time.Millisecond) // Даем время writePump запуститься
		select {
		case client.send <- welcomeMsg:
			log.Printf("WELCOME message queued for client %s", clientID)
		case <-time.After(1 * time.Second):
			log.Printf("WARNING: Failed to send WELCOME message to client %s (channel full or closed)", clientID)
		}
	}()

	select {}
}

// Цикл чтения из WebSocket
func readPump(client *WebSocketClient) {
	defer func() {
		log.Printf("readPump exiting for client %s", client.clientID)
	}()

	client.conn.SetReadDeadline(time.Now().Add(70 * time.Second))
	client.conn.SetPongHandler(func(string) error {
		log.Printf("Received PONG from client %s", client.clientID)
		client.conn.SetReadDeadline(time.Now().Add(70 * time.Second))
		return nil
	})

	log.Printf("readPump started for client %s, waiting for messages...", client.clientID)

	for {
		var msg WebSocketMessage
		err := client.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error for %s: %v", client.clientID, err)
			} else if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Printf("WebSocket closed normally for %s: %v", client.clientID, err)
			} else {
				log.Printf("WebSocket read error for %s: %v (type: %T)", client.clientID, err, err)
			}
			break
		}

		log.Printf("Received from %s: %s", client.clientID, msg.Type)

		switch msg.Type {
		case "PING":
			client.send <- WebSocketMessage{
				Type:      "PONG",
				ClientID:  client.clientID,
				Timestamp: time.Now().Unix(),
			}

		case "FRAME":
			payloadBytes, err := json.Marshal(msg.Payload)
			if err != nil {
				log.Printf("Failed to marshal payload: %v", err)
				client.send <- WebSocketMessage{
					Type: "ERROR",
					Payload: map[string]interface{}{
						"message": "Invalid payload format",
					},
				}
				continue
			}

			var frameData models.WSFrameMessage
			if err := json.Unmarshal(payloadBytes, &frameData); err != nil {
				log.Printf("Invalid frame data format: %v", err)
				client.send <- WebSocketMessage{
					Type: "ERROR",
					Payload: map[string]interface{}{
						"message": "Invalid frame data format",
					},
				}
				continue
			}

			frameBytes, err := base64.StdEncoding.DecodeString(frameData.Frame)
			if err != nil {
				log.Printf("Base64 decode error: %v", err)
				client.send <- WebSocketMessage{
					Type: "ERROR",
					Payload: map[string]interface{}{
						"message": "Invalid base64",
					},
				}
				continue
			}

			if len(frameBytes) == 0 {
				log.Printf("Empty frame data from client %s", client.clientID)
				continue
			}

			videoFrame := &pb.VideoFrame{
				FrameData:      frameBytes,
				Timestamp:      frameData.Timestamp,
				SequenceNumber: frameData.SequenceNumber,
			}

			if grpcClient == nil {
				client.send <- WebSocketMessage{
					Type: "ERROR",
					Payload: map[string]interface{}{
						"message": "ML service unavailable",
					},
				}
				continue
			}

			if !grpcClient.HealthCheck() {
				client.send <- WebSocketMessage{
					Type: "ERROR",
					Payload: map[string]interface{}{
						"message": "ML service disconnected",
					},
				}
				continue
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			result, err := grpcClient.ProcessFrame(ctx, videoFrame)
			cancel()

			if err != nil {
				log.Printf("gRPC error: %v", err)
				client.send <- WebSocketMessage{
					Type: "ERROR",
					Payload: map[string]interface{}{
						"message": "Processing failed",
					},
				}
				continue
			}
			resp := WebSocketMessage{
				Type:      "DETECTION_RESULT",
				ClientID:  client.clientID,
				Timestamp: time.Now().Unix(),
				Payload: map[string]interface{}{
					"is_drowsy":        result.IsDrowsy,
					"drowsiness_score": result.DrowsinessScore,
					"alert_level":      result.AlertLevel,
					"inference_time":   result.InferenceTimeMs,
					"sequence_number":  frameData.SequenceNumber,
				},
			}
			client.send <- resp

		default:
			log.Printf("Unknown message type: %s", msg.Type)
		}
	}
}

func writePump(client *WebSocketClient) {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		log.Printf("writePump exiting for client %s", client.clientID)
		ticker.Stop()
	}()

	log.Printf("writePump started for client %s, ready to send messages...", client.clientID)

	for {
		select {
		case msg, ok := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

			if !ok {
				log.Printf("Send channel closed for client %s", client.clientID)
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			log.Printf("Attempting to send message type %s to client %s", getMessageType(msg), client.clientID)
			if err := client.conn.WriteJSON(msg); err != nil {
				log.Printf("Write error for client %s: %v", client.clientID, err)
				return
			}
			log.Printf("Successfully sent message type %s to client %s", getMessageType(msg), client.clientID)

		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Ping error for client %s: %v", client.clientID, err)
				return
			}
			log.Printf("Sent PING to client %s", client.clientID)
		}
	}
}

func getMessageType(msg interface{}) string {
	if wsMsg, ok := msg.(WebSocketMessage); ok {
		return wsMsg.Type
	}
	return "unknown"
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	enableCORS(w, r, appConfig)

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

func handleMetrics(w http.ResponseWriter, r *http.Request) {
	enableCORS(w, r, appConfig)
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

func generateClientID() string {
	return "client-" + time.Now().Format("20060102150405")
}

func closeAllWebSocketConnections() {
	wsClients.mu.Lock()
	defer wsClients.mu.Unlock()

	for clientID, client := range wsClients.clients {
		if atomic.CompareAndSwapInt32(&client.closed, 0, 1) {
			close(client.send)
		}
		client.conn.Close()
		log.Printf("Closed connection for client: %s", clientID)
	}
	wsClients.clients = make(map[string]*WebSocketClient)
}
