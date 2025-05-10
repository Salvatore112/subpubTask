package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"strconv"
	"subpub_demo/config"
	"subpub_demo/proto"
	"subpub_demo/server"
	"subpub_demo/subpub"
	"syscall"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Setup logger
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetFormatter(&logrus.JSONFormatter{})

	// Load config
	cfg := config.DefaultConfig()
	logger.WithField("config", cfg).Info("Using configuration")

	// Create pubsub bus
	bus := subpub.NewSubPub()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), cfg.GRPC.ShutdownTimeout)
		defer cancel()
		if err := bus.Close(ctx); err != nil {
			logger.WithError(err).Error("Failed to gracefully shutdown pubsub bus")
		}
	}()

	// Create gRPC server
	grpcServer := grpc.NewServer()
	reflection.Register(grpcServer)
	service := server.NewPubSubService(bus, logger)
	proto.RegisterPubSubServer(grpcServer, service)

	// Start server

	lis, err := net.Listen("tcp", net.JoinHostPort("", strconv.Itoa(cfg.GRPC.Port)))
	if err != nil {
		logger.WithError(err).Fatal("Failed to listen")
	}

	go func() {
		logger.WithField("port", cfg.GRPC.Port).Info("Starting gRPC server")
		if err := grpcServer.Serve(lis); err != nil {
			logger.WithError(err).Fatal("gRPC server failed")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.GRPC.ShutdownTimeout)
	defer cancel()

	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		logger.Info("Server stopped gracefully")
	case <-ctx.Done():
		logger.Warn("Forcing server shutdown after timeout")
		grpcServer.Stop()
	}
}
