package handlers

import (
	"AI_DETECTOR/go-backend/internal/services"
	pb "AI_DETECTOR/go-backend/pkg/pb"
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"log"
	"time"
)

type GRPCHandler struct {
	pb.UnimplementedDrowsinessDetectionServer
	grpcClient *services.GRPCClient
	metrics    *services.Metrics
}

func NewGRPCHandler(grpcClient *services.GRPCClient) *GRPCHandler {
	return &GRPCHandler{
		grpcClient: grpcClient,
		metrics:    services.NewMetrics(),
	}
}

func (h *GRPCHandler) DetectDrowsiness(ctx context.Context, req *pb.VideoFrame) (*pb.DetectionResult, error) {
	start := time.Now()

	if req.FrameData == nil || len(req.FrameData) == 0 {
		return nil, status.Error(codes.InvalidArgument, "frame_data is required")
	}

	if h.grpcClient == nil {
		return nil, status.Error(codes.Unavailable, "grpc client is nil")
	}

	log.Printf("Frame #%d, size: %d bytes", req.SequenceNumber, len(req.FrameData))

	result, err := h.grpcClient.ProcessFrame(ctx, req)
	if err != nil {
		log.Printf("Error: %v", err)
		h.metrics.IncrementErrors()
		return nil, status.Error(codes.Internal, "processing failed")
	}

	duration := time.Since(start)
	h.metrics.RecordLatency(duration)
	h.metrics.IncrementFrames()

	log.Printf("Frame #%d processed in %v, drowsy: %v", req.SequenceNumber, duration, result.IsDrowsy)
	return result, nil
}

func (h *GRPCHandler) DetectDrowsinessStream(stream pb.DrowsinessDetection_DetectDrowsinessStreamServer) error {
	log.Println("Stream started")

	pythonStream, err := h.grpcClient.StartStream(stream.Context())
	if err != nil {
		log.Printf("Failed to start Python stream: %v", err)
		return status.Error(codes.Internal, "starting Python stream failed")
	}

	errChan := make(chan error, 2)

	// Горутина 1: Клиент -> Python
	go func() {
		for {
			req, err := stream.Recv()
			if err == io.EOF {
				pythonStream.CloseSend()
				errChan <- nil
				return
			}
			if err != nil {
				log.Printf("Recv error: %v", err)
				errChan <- err
				return
			}
			if err := pythonStream.Send(req); err != nil {
				log.Printf("Send error: %v", err)
				errChan <- err
				return
			}

			h.metrics.IncrementFrames()
		}
	}()

	// Горутина 2: Python -> Клиент
	go func() {
		for {
			result, err := pythonStream.Recv()
			if err == io.EOF {
				errChan <- nil
				return
			}
			if err != nil {
				log.Printf("Python recv error: %v", err)
				errChan <- err
				return
			}
			if err := stream.Send(result); err != nil {
				log.Printf("Client send error: %v", err)
				errChan <- err
				return
			}
		}
	}()

	err = <-errChan
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	log.Println("Stream completed")
	return nil
}

func (h *GRPCHandler) Health(ctx context.Context, _ *pb.Empty) (*pb.HealthStatus, error) {
	pythonHealthy := false
	if h.grpcClient != nil {
		pythonHealthy = h.grpcClient.HealthCheck()
	}

	log.Printf("Health: Python=%v, Clients=%d", pythonHealthy, h.metrics.GetActiveClients())

	return &pb.HealthStatus{
		Status:        "healthy",
		GrpcService:   pythonHealthy,
		ActiveClients: int32(h.metrics.GetActiveClients()),
	}, nil
}
