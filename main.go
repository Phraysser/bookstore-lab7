package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"example.com/bookstore/book"
	"example.com/bookstore/endpoint"
	"example.com/bookstore/pb"
	"example.com/bookstore/repository"
	"example.com/bookstore/transport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	httpAddr := flag.String("http.addr", ":8080", "HTTP listen address")
	grpcAddr := flag.String("grpc.addr", ":9090", "gRPC listen address")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, *httpAddr, *grpcAddr); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, httpAddr, grpcAddr string) error {
	httpListener, err := net.Listen("tcp", httpAddr)
	if err != nil {
		return fmt.Errorf("listen HTTP: %w", err)
	}
	defer httpListener.Close()

	grpcListener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return fmt.Errorf("listen gRPC: %w", err)
	}
	defer grpcListener.Close()

	endpoints := endpoint.New(book.NewService(repository.NewMemory()))
	httpServer := &http.Server{
		Handler: transport.NewHTTP(endpoints), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second,
	}
	grpcServer := grpc.NewServer()
	pb.RegisterBookServiceServer(grpcServer, transport.NewGRPC(endpoints))
	reflection.Register(grpcServer)

	serverErrors := make(chan error, 2)
	go func() { serverErrors <- httpServer.Serve(httpListener) }()
	go func() { serverErrors <- grpcServer.Serve(grpcListener) }()
	log.Printf("HTTP %s | gRPC %s", httpListener.Addr(), grpcListener.Addr())

	var serveErr error
	select {
	case serveErr = <-serverErrors:
	case <-ctx.Done():
	}

	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	grpcDone := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(grpcDone)
	}()
	if err := httpServer.Shutdown(shutdown); err != nil {
		_ = httpServer.Close()
	}
	select {
	case <-grpcDone:
	case <-shutdown.Done():
		grpcServer.Stop()
	}
	if errors.Is(serveErr, http.ErrServerClosed) || errors.Is(serveErr, grpc.ErrServerStopped) {
		return nil
	}
	return serveErr
}
