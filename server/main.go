package main

import (
	"log"
	"net"
	"os"

	pb "github.com/Aditya-Chowdhary/gRPC-Go/proto/todo/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	args := os.Args[1:]
	var addr string
	if len(args) == 0 {
		// log.Fatalln("usage: client [IP_ADDR]")
		addr = "0.0.0.0:50051"
	} else {
		addr = args[0]
	}
	lis, err := net.Listen("tcp", addr)

	if err != nil {
		log.Fatalf("failed to listen: %v\n", err)
	}

	defer func(lis net.Listener) {
		if err := lis.Close(); err != nil {
			log.Fatalf("unexpected error: %v", err)
		}
	}(lis)

	creds, err := credentials.NewServerTLSFromFile("./certs/server_cert.pem", "./certs/server_key.pem")
	if err != nil {
		log.Fatalf("failed to create credentials: %v", err)
	}

	log.Printf("listening at %s\n", addr)

	opts := []grpc.ServerOption{
		grpc.Creds(creds),
		grpc.UnaryInterceptor(unaryAuthInterceptor),
		grpc.StreamInterceptor(streamAuthInterceptor),
	}
	s := grpc.NewServer(opts...)

	pb.RegisterTodoServiceServer(s, &server{
		d: New(),
	})

	defer s.Stop()
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v\n", err)
	}
}
