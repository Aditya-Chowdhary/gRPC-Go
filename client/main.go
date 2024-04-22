package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	pb "github.com/Aditya-Chowdhary/gRPC-Go/proto/todo/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	conn, err := grpc.Dial(addr, opts...)

	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	c := pb.NewTodoServiceClient(conn)

	fmt.Println("-------ADD--------")
	dueDate := time.Now().Add(5 * time.Second)
	addTask(c, "This is a task", dueDate)
	fmt.Println("------------------")

	fmt.Println("-------List--------")
	printTasks(c)
	fmt.Println("------------------")

	defer func(conn *grpc.ClientConn) {
		if err := conn.Close(); err != nil {
			log.Fatalf("unexpected error: %v", err)
		}
	}(conn)
}

func addTask(c pb.TodoServiceClient, description string, dueDate time.Time) uint64 {
	req := &pb.AddTaskRequest{
		Description: description,
		DueDate:     timestamppb.New(dueDate),
	}
	res, err := c.AddTask(context.Background(), req)
	if err != nil {
		panic(err)
	}
	fmt.Printf("added task: %d\n", res.Id)
	return res.Id
}

func printTasks(c pb.TodoServiceClient) {
	req := &pb.ListTasksRequest{}
	stream, err := c.ListTasks(context.Background(), req)
	if err != nil {
		log.Fatalf("unexpected error: %v", err)
	}

	for {
		res, err := stream.Recv()
		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatalf("unexpected error: %v", err)
		}

		fmt.Println(res.Task.String(), "overdue: ", res.Overdue)
	}
}
