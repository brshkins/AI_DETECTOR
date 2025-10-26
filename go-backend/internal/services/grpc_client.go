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
			Time:                10 * time.Second,
			Timeout:             3 * time.Second,
			PermitWithoutStream: true,
		}),
	}

	conn, err := grpc.Dial(url, opts...)
	if err != nil {
		return nil, fmt.Errorf("could not connect to Python gRPC server at %s: %s", url, err)
	}

	client := pb.NewDrowsinessDetectionClient(conn)
	log.Printf("Connected to Python gRPC server at %s", url)

	return &GRPCClient{
		conn:   conn,
		client: client,
		url:    url,
	}, nil
}

func (gc *GRPCClient) ProcessFrame(ctx context.Context, frame *pb.VideoFrame) (*pb.DetectionResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := gc.client.DetectDrowsiness(ctx, frame)
	if err != nil {
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
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := gc.client.Health(ctx, &pb.Empty{})
	return err == nil
}

func (gc *GRPCClient) Close() error {
	if gc.conn != nil {
		return gc.conn.Close()
	}
	return nil
}
