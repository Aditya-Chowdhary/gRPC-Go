package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	pb "github.com/Aditya-Chowdhary/gRPC-Go/proto/todo/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

func printTasks(c pb.TodoServiceClient, fm *fieldmaskpb.FieldMask) {
	req := &pb.ListTasksRequest{
		Mask: fm,
	}
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

func updateTask(c pb.TodoServiceClient, reqs ...*pb.UpdateTasksRequest) {
	stream, err := c.UpdateTasks(context.Background())
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
