package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/segmentio/kafka-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	pb "github.com/greenlane/ingestion/proto"
)

const (
	grpcPort      = ":50051"
	redisAddr     = "localhost:6379"
	kafkaBroker   = "localhost:19092"
	kafkaTopic    = "fleet-events"
	apiTokenValue = "greenlane-secret-token"
)

type IngestionServer struct {
	pb.UnimplementedFleetServiceServer
	redisClient *redis.Client
	kafkaWriter *kafka.Writer
}

func main() {
	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "",
		DB:       0,
	})

	// Test Redis connection
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("ERROR: Failed to connect with Redis: %v", err)
	}
	log.Println("INFO: Connected to Redis")

	// Initialize Kafka writer
	kafkaWriter := &kafka.Writer{
		Addr:                   kafka.TCP(kafkaBroker),
		Topic:                  kafkaTopic,
		Balancer:               &kafka.LeastBytes{},
		AllowAutoTopicCreation: true,
		Compression:            kafka.Snappy,
	}

	// Test Kafka connection by creating topic
	conn, err := kafka.Dial("tcp", kafkaBroker)
	if err != nil {
		log.Fatalf("ERROR: Failed to connect to Kafka: %v", err)
	}
	defer conn.Close()
	log.Println("SUCCESS: Connected to Redpanda (Kafka)")

	// Create gRPC server with auth interceptor
	server := grpc.NewServer(
		grpc.UnaryInterceptor(authUnaryInterceptor),
		grpc.StreamInterceptor(authStreamInterceptor),
	)

	ingestionServer := &IngestionServer{
		redisClient: redisClient,
		kafkaWriter: kafkaWriter,
	}

	pb.RegisterFleetServiceServer(server, ingestionServer)

	// Start listening
	listener, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("ERROR: Failed to listen: %v", err)
	}

	log.Printf("SUCCESS: GreenLane Ingestion Service started on %s", grpcPort)
	log.Println("INFO: Waiting for EV telemetry streams...")

	// Graceful shutdown
	go func() {
		if err := server.Serve(listener); err != nil {
			log.Fatalf("ERROR: Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("INFO: Shutting down gracefully...")
	server.GracefulStop()
	redisClient.Close()
	kafkaWriter.Close()
}

// StreamTelemetry handles bidirectional streaming of car telemetry
func (s *IngestionServer) StreamTelemetry(stream pb.FleetService_StreamTelemetryServer) error {
	log.Println("INFO: New telemetry stream connected")

	for {
		carStatus, err := stream.Recv()
		if err != nil {
			log.Printf("ERROR: Stream ended: %v", err)
			return err
		}

		log.Printf("📍 Received telemetry from Car %s: Lat=%.4f, Lon=%.4f, Battery=%.1f%%",
			carStatus.CarId, carStatus.Latitude, carStatus.Longitude, carStatus.BatteryLevel)

		// Write to Redis (Geospatial)
		ctx := context.Background()
		if err := s.writeToRedis(ctx, carStatus); err != nil {
			log.Printf("⚠️  Redis write failed: %v", err)
		}

		// Emit to Kafka/Redpanda
		if err := s.emitToKafka(ctx, carStatus); err != nil {
			log.Printf("⚠️  Kafka emit failed: %v", err)
		}

		// Send acknowledgment (optional response)
		response := &pb.BookingResponse{
			BookingId: fmt.Sprintf("ack-%s-%d", carStatus.CarId, time.Now().UnixMilli()),
			CarId:     carStatus.CarId,
			Status:    pb.BookingStatus_BOOKING_UNKNOWN,
			Message:   "Telemetry received",
			Timestamp: time.Now().UnixMilli(),
		}

		if err := stream.Send(response); err != nil {
			log.Printf("ERROR: Failed to send response: %v", err)
			return err
		}
	}
}

// writeToRedis stores car location in Redis Geospatial index
func (s *IngestionServer) writeToRedis(ctx context.Context, carStatus *pb.CarStatus) error {
	key := "fleet:locations"

	// GEOADD key longitude latitude member
	_, err := s.redisClient.GeoAdd(ctx, key, &redis.GeoLocation{
		Name:      carStatus.CarId,
		Longitude: carStatus.Longitude,
		Latitude:  carStatus.Latitude,
	}).Result()

	if err != nil {
		return fmt.Errorf("ERROR: GEOADD failed: %w", err)
	}

	// Also store battery level and timestamp
	s.redisClient.HSet(ctx, fmt.Sprintf("car:%s", carStatus.CarId), map[string]interface{}{
		"battery":   carStatus.BatteryLevel,
		"velocity":  carStatus.Velocity,
		"timestamp": carStatus.Timestamp,
	})

	return nil
}

// emitToKafka sends telemetry event to Redpanda
func (s *IngestionServer) emitToKafka(ctx context.Context, carStatus *pb.CarStatus) error {
	message := kafka.Message{
		Key: []byte(carStatus.CarId),
		Value: []byte(fmt.Sprintf(`{"car_id":"%s","lat":%.6f,"lon":%.6f,"battery":%.2f,"velocity":%.2f,"timestamp":%d,"event_type":"telemetry"}`,
			carStatus.CarId,
			carStatus.Latitude,
			carStatus.Longitude,
			carStatus.BatteryLevel,
			carStatus.Velocity,
			carStatus.Timestamp,
		)),
		Time: time.Now(),
	}

	return s.kafkaWriter.WriteMessages(ctx, message)
}

// authUnaryInterceptor validates API token for unary calls
func authUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if err := validateToken(ctx); err != nil {
		return nil, err
	}
	return handler(ctx, req)
}

// authStreamInterceptor validates API token for streaming calls
func authStreamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	if err := validateToken(ss.Context()); err != nil {
		return err
	}
	return handler(srv, ss)
}

// validateToken checks for x-api-token in metadata
func validateToken(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "missing metadata")
	}

	tokens := md.Get("x-api-token")
	if len(tokens) == 0 {
		return status.Error(codes.Unauthenticated, "missing x-api-token")
	}

	if tokens[0] != apiTokenValue {
		return status.Error(codes.Unauthenticated, "invalid x-api-token")
	}

	return nil
}
