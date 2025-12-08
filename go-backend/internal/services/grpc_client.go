package services

import (
	pb "AI_DETECTOR/go-backend/pkg/pb"
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"log"
	"time"
)

type GRPCClient struct {
	conn   *grpc.ClientConn
	client pb.DrowsinessDetectionClient
	url    string
}

func NewGRPCClient(url string) (*GRPCClient, error) {
	log.Printf("Connecting to Python gRPC at %s", url)

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(50*1024*1024),
			grpc.MaxCallSendMsgSize(50*1024*1024),
		),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: false,
		}),
	}

	opts = append(opts, grpc.WithBlock())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, url, opts...)
	if err != nil {
		return nil, fmt.Errorf("could not connect to Python gRPC server at %s: %s", url, err)
	}

	ctxReady, cancelReady := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelReady()

	for {
		state := conn.GetState()
		if state.String() == "READY" {
			break
		}
		if !conn.WaitForStateChange(ctxReady, state) {
			conn.Close()
			return nil, fmt.Errorf("connection to %s did not become READY (state: %s)", url, state.String())
		}
	}

	client := pb.NewDrowsinessDetectionClient(conn)
	log.Printf("Connected to Python gRPC server at %s (state: %s)", url, conn.GetState().String())

	return &GRPCClient{
		conn:   conn,
		client: client,
		url:    url,
	}, nil
}

func (gc *GRPCClient) ProcessFrame(ctx context.Context, frame *pb.VideoFrame) (*pb.DetectionResult, error) {
	if gc == nil || gc.client == nil {
		return nil, fmt.Errorf("gRPC client is not initialized")
	}

	state := gc.conn.GetState()
	if state.String() != "READY" {
		return nil, fmt.Errorf("gRPC connection not ready (state: %s)", state.String())
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := gc.client.DetectDrowsiness(ctx, frame)
	if err != nil {
		log.Printf("ProcessFrame error: %v (connection state: %s)", err, gc.conn.GetState().String())
		return nil, fmt.Errorf("could not detect drowsiness: %w", err)
	}
	return result, nil
}

func (gc *GRPCClient) StartStream(ctx context.Context) (pb.DrowsinessDetection_DetectDrowsinessStreamClient, error) {
	stream, err := gc.client.DetectDrowsinessStream(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not detect drowsiness stream: %w", err)
	}
	return stream, nil
}

func (gc *GRPCClient) HealthCheck() bool {
	if gc == nil || gc.client == nil || gc.conn == nil {
		return false
	}

	state := gc.conn.GetState()
	if state.String() != "READY" {
		log.Printf("gRPC connection state: %s (not READY)", state.String())
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := gc.client.Health(ctx, &pb.Empty{})
	if err != nil {
		log.Printf("Health check failed: %v (connection state: %s)", err, gc.conn.GetState().String())
		return false
	}
	return true
}

func (gc *GRPCClient) Close() error {
	if gc.conn != nil {
		return gc.conn.Close()
	}
	return nil
}
