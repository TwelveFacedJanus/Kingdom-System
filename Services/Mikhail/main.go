package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	authenticate "Mikhail/gen/proto"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	port = getEnv("MIKHAIL_PORT", "50051")
)

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Sync()
	sugar := logger.Sugar()

	// Create context that listens for the interrupt signal from the OS
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Create listener
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		sugar.Fatalf("failed to listen: %v", err)
	}

	// Create gRPC server with interceptors
	s := grpc.NewServer(
		grpc.UnaryInterceptor(loggingInterceptor(sugar)),
	)

	// Create and register auth server
	authServer := NewAuthServer()
	authenticate.RegisterAuthenticateServiceServer(s, authServer)

	// Register reflection service on gRPC server
	reflection.Register(s)

	// Start server in a goroutine
	go func() {
		sugar.Infof("server listening at %v", lis.Addr())
		if err := s.Serve(lis); err != nil {
			sugar.Fatalf("failed to serve: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-ctx.Done()

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initiate graceful shutdown
	sugar.Info("shutting down server...")

	// Stop accepting new connections
	if err := lis.Close(); err != nil {
		sugar.Errorf("failed to close listener: %v", err)
	}

	// Create a channel to signal when shutdown is complete
	shutdownComplete := make(chan struct{})

	// Graceful shutdown in a goroutine
	go func() {
		s.GracefulStop()
		close(shutdownComplete)
	}()

	// Wait for either shutdown to complete or timeout
	select {
	case <-shutdownComplete:
		sugar.Info("server stopped gracefully")
	case <-shutdownCtx.Done():
		sugar.Warn("shutdown timed out, forcing stop")
		s.Stop()
	}

	// Close auth server resources
	if err := authServer.Close(); err != nil {
		sugar.Errorf("failed to close auth server: %v", err)
	}
}

// loggingInterceptor returns a new unary server interceptor for logging
func loggingInterceptor(logger *zap.SugaredLogger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(start)

		if err != nil {
			logger.Errorw("request failed",
				"method", info.FullMethod,
				"duration", duration,
				"error", err,
			)
		} else {
			logger.Infow("request completed",
				"method", info.FullMethod,
				"duration", duration,
			)
		}

		return resp, err
	}
}
