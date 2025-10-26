package main

import (
	"AI_DETECTOR/go-backend/internal/config"
	"AI_DETECTOR/go-backend/internal/handlers"
	"AI_DETECTOR/go-backend/internal/services"
	"AI_DETECTOR/go-backend/pkg/pb"
	"context"
	"flag"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	grpcPort := flag.String("grpc-port", ":50051", "gRPC port")
	pythonURL := flag.String("python-url", "localhost:50052", "Python service URL")
	flag.Parse()

	cfg := config.LoadConfig()

	log.Println("Starting...")
	log.Printf("gRPC port: %s", *grpcPort)
	log.Printf("Python service: %s", *pythonURL)
	log.Printf("Enviroment: %s", cfg.Environment)

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Подключение к Python
	grpcClient, err := services.NewGRPCClient(*pythonURL)
	if err != nil {
		log.Printf("Python service unavailable: %v", err)
		log.Println("Continuing without Python (for testing)")
	}

	if grpcClient != nil {
		defer grpcClient.Close()
	}

	// gRPC сервер
	s := grpc.NewServer(
		grpc.MaxRecvMsgSize(50*1024*1024),
		grpc.MaxSendMsgSize(50*1024*1024),
	)
	grpcHandler := handlers.NewGRPCHandler(grpcClient)
	pb.RegisterDrowsinessDetectionServer(s, grpcHandler)

	// Запуск в горутине
	go func() {
		lis, err := net.Listen("tcp", ":"+*grpcPort)
		if err != nil {
			log.Fatalf("Failed to listen: %v", err)
		}

		log.Printf("gRPC server listening on port %s", *grpcPort)
		log.Println("Waiting for connections...")

		if err := s.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Ждём сигнала
	<-done
	log.Println("Shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stopped := make(chan struct{})
	go func() {
		s.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		log.Println("Stopped")
	case <-shutdownCtx.Done():
		log.Println("Forced shutdown")
		s.Stop()
	}

	log.Println("Goodbye!")
}
