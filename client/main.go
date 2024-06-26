package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	pb "github.com/Aditya-Chowdhary/gRPC-Go/proto/todo/v2"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
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

	_, err := credentials.NewClientTLSFromFile("./certs/ca_cert.pem", "x.test.example.com")
	if err != nil {
		log.Fatalf("failed to load credentials: %v", err)
	}

	retryOpts := []retry.CallOption{
		retry.WithMax(3),
		retry.WithBackoff(retry.BackoffExponential(100 * time.Millisecond)),
		retry.WithCodes(codes.Unavailable),
	}

	opts := []grpc.DialOption{
		// grpc.WithTransportCredentials(creds),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(
			retry.UnaryClientInterceptor(retryOpts...),
			unaryAuthInterceptor),
		grpc.WithStreamInterceptor(streamAuthInterceptor),
	}

	conn, err := grpc.Dial(addr, opts...)

	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	c := pb.NewTodoServiceClient(conn)

	fmt.Println("-------ADD--------")
	dueDate := time.Now().Add(5 * time.Second)
	addTask(c, "This is a task", dueDate)
	addTask(c, "This is another task", dueDate)
	addTask(c, "This is one more task", dueDate)
	fmt.Println("------------------")

	fm, err := fieldmaskpb.New(&pb.Task{}, "id")
	if err != nil {
		log.Fatalf("unexpected error: %v", err)
	}
	fmt.Println("-------List--------")
	printTasks(c, fm)
	fmt.Println("------------------")

	fmt.Println("------Update-------")
	updates := []*pb.UpdateTasksRequest{
		{Id: 1, Description: "This is actually task 1"},
		{Id: 2, DueDate: timestamppb.New(dueDate.Add(5 * time.Hour))},
		{Id: 3, Done: true},
	}
	updateTask(c, updates...)
	printTasks(c, nil)
	fmt.Println("------------------")

	fmt.Println("---------DELETE----------")
	deleteTasks(c, []*pb.DeleteTasksRequest{
		{Id: 1},
		{Id: 2},
		{Id: 3},
	}...)
	printTasks(c, nil)
	fmt.Println("------------------")

	// fmt.Println("--------ERROR---------")
	// addTask(c, "", dueDate)
	// fmt.Println("----------------------")

	// fmt.Println("--------ERROR---------")
	// addTask(c, "notEmpty", time.Now().Add(-5*time.Second))
	// fmt.Println("----------------------")

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
		if s, ok := status.FromError(err); ok {
			switch s.Code() {
			case codes.InvalidArgument, codes.Internal:
				log.Fatalf("%s: %s", s.Code(), s.Message())
			default:
				log.Fatal(s)
			}
		} else {
			panic(err)
		}
	}
	fmt.Printf("added task: %d\n", res.Id)
	return res.Id
}

func printTasks(c pb.TodoServiceClient, fm *fieldmaskpb.FieldMask) {
	// ctx, cancel := context.WithCancel(context.Background())
	// ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	// defer cancel()
	ctx := context.Background()

	req := &pb.ListTasksRequest{
		Mask: fm,
	}
	stream, err := c.ListTasks(ctx, req)
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

		// if res.Overdue {
		// 	log.Println("CANCEL called")
		// 	cancel()
		// }

		fmt.Println(res.Task.String(), "overdue: ", res.Overdue)
	}
}

func updateTask(c pb.TodoServiceClient, reqs ...*pb.UpdateTasksRequest) {
	ctx := context.Background()

	stream, err := c.UpdateTasks(ctx)
	if err != nil {
		log.Fatalf("unexpected error: %v", err)
	}

	for _, req := range reqs {
		err := stream.Send(req)
		if err != nil {
			log.Fatalf("unexpected error: %v", err)
			return
		}

		if req != nil {
			fmt.Printf("updated task with id: %d\n", req.Id)
		}

	}

	if _, err := stream.CloseAndRecv(); err != nil {
		log.Fatalf("unexpected error: %v", err)
	}
}

func deleteTasks(c pb.TodoServiceClient, reqs ...*pb.DeleteTasksRequest) {
	stream, err := c.DeleteTasks(context.Background())

	if err != nil {
		log.Fatalf("unexpected error: %v", err)
	}

	waitc := make(chan struct{})

	go func() {
		for {
			_, err := stream.Recv()

			if err == io.EOF {
				close(waitc)
				break
			}
			if err != nil {
				log.Fatalf("error while receiving: %v\n", err)
			}

			log.Println("deleted tasks")
		}
	}()

	for _, req := range reqs {
		if err := stream.Send(req); err != nil {
			return
		}
	}
	if err := stream.CloseSend(); err != nil {
		return
	}

	<-waitc
}
