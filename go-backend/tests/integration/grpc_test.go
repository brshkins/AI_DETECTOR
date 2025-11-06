package integration

import (
	"AI_DETECTOR/go-backend/pkg/pb"
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"testing"
	"time"
)

func TestGRPCDetectDrowsiness(t *testing.T) {
	// Подключение
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewDrowsinessDetectionClient(conn)

	// Запрос
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &pb.VideoFrame{
		FrameData:      []byte("test frame data"),
		Timestamp:      time.Now().UnixMilli(),
		SequenceNumber: 1,
	}

	// Вызов
	result, err := client.DetectDrowsiness(ctx, req)
	if err != nil {
		t.Fatalf("DetectDrowsiness failed: %v", err)
	}

	// Проверка
	if result == nil {
		t.Fatalf("Result is nil")
	}

	t.Logf("Success! drowsy=%v, score=%.2f", result.IsDrowsy, result.DrowsinessScore)
}

func TestGRPCHealth(t *testing.T) {
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewDrowsinessDetectionClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	status, err := client.Health(ctx, &pb.Empty{})
	if err != nil {
		t.Fatalf("Health failed: %v", err)
	}

	if status.Status != "healthy" {
		t.Errorf("Expected healthy, got %s", status.Status)
	}

	t.Logf("Health: %+v", status)
}
