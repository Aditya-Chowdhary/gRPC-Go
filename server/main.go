package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/Aditya-Chowdhary/gRPC-Go/proto/todo/v2"
	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func newMetricsServer(httpAddr string, reg *prometheus.Registry) *http.Server {
	httpSrv := &http.Server{Addr: httpAddr}
	m := http.NewServeMux()
	m.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	httpSrv.Handler = m
	return httpSrv
}

func newGrpcServer(lis net.Listener, srvMetric *grpcprom.ServerMetrics) (*grpc.Server, error) {
	_, err := credentials.NewServerTLSFromFile("./certs/server_cert.pem", "./certs/server_key.pem")
	if err != nil {
		return nil, err
	}

	logger := log.New(os.Stderr, "", log.Ldate|log.Ltime)

	opts := []grpc.ServerOption{
		// grpc.Creds(creds),
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			// otelgrpc.UnaryServerInterceptor(),
			srvMetric.UnaryServerInterceptor(),
			auth.UnaryServerInterceptor(validateAuthToken),
			logging.UnaryServerInterceptor(logCalls(logger)),
		),
		grpc.ChainStreamInterceptor(
			// otelgrpc.StreamServerInterceptor(),
			srvMetric.StreamServerInterceptor(),
			auth.StreamServerInterceptor(validateAuthToken),
			logging.StreamServerInterceptor(logCalls(logger)),
		),
	}
	s := grpc.NewServer(opts...)

	pb.RegisterTodoServiceServer(s, &server{
		d: New(),
	})

	return s, nil
}

func main() {
	args := os.Args[1:]
	var grpcAddr string
	var httpAddr string
	if len(args) != 2 {
		// log.Fatalln("usage: client [IP_ADDR]")
		grpcAddr = "0.0.0.0:50051"
		httpAddr = "0.0.0.0:50052"
	} else if len(args) == 2 {
		grpcAddr = args[0]
		httpAddr = args[1]
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(quit)

	srvMetrics := grpcprom.NewServerMetrics(
		grpcprom.WithServerHandlingTimeHistogram(
			grpcprom.WithHistogramBuckets([]float64{0.001, 0.01, 0.1, 0.3, 0.6, 1, 3, 6, 9, 20, 30, 60, 90, 120}),
		),
	)
	reg := prometheus.NewRegistry()
	reg.MustRegister(srvMetrics)

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v\n", err)
	}

	g, ctx := errgroup.WithContext(ctx)
	grpcServer, err := newGrpcServer(lis, srvMetrics)
	if err != nil {
		log.Fatalf("failed to make GrpcServer: %v\n", err)
	}

	g.Go(func() error {
		log.Printf("gRPC server listening at %s\n", grpcAddr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("failed to gRPC server: %v\n", err)
			return err
		}
		log.Println("gRPC server shutdown")
		return nil
	})

	metricsServer := newMetricsServer(httpAddr, reg)
	g.Go(func() error {
		log.Printf("metrics server listening at %s\n", httpAddr)
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed{
			log.Printf("failed to serve metrics: %v\n", err)
			return err
		}
		log.Println("metrics server shutdown")
		return nil
	})

	select {
	case <-quit:
		break
	case <-ctx.Done():
		break
	}

	cancel()

	timeOutctx, timoutCancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer timoutCancel()

	log.Println("Shutting down servers...")

	grpcServer.GracefulStop()
	metricsServer.Shutdown(timeOutctx)

	if err := g.Wait(); err != nil {
		log.Fatal("error: ", err)
	}
}
