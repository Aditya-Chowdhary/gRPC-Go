package main

import (
	"context"
	"io"
	"log"
	"slices"
	"time"

	pb "github.com/Aditya-Chowdhary/gRPC-Go/proto/todo/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func (s *server) AddTask(_ context.Context, in *pb.AddTaskRequest) (*pb.AddTaskResponse, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	id, err := s.d.addTask(in.Description, in.DueDate.AsTime())
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			"unexpected error: %s", err.Error(),
		)
	}

	return &pb.AddTaskResponse{Id: id}, nil
}

func (s *server) ListTasks(req *pb.ListTasksRequest, stream pb.TodoService_ListTasksServer) error {
	ctx := stream.Context()

	return s.d.getTasks(func(t interface{}) error {
		select {
		case <-ctx.Done():
			switch ctx.Err() {
			case context.Canceled:
				log.Printf("request canceled: %s\n", ctx.Err())
			case context.DeadlineExceeded:
				log.Printf("request deadline exceeded: %s\n", ctx.Err())
			}
			return ctx.Err()
		case <-time.After(1 * time.Millisecond):
		}

		task := t.(*pb.Task)

		Filter(task, req.Mask)

		overdue := task.DueDate != nil && !task.Done && task.DueDate.AsTime().Before(time.Now().UTC())
		err := stream.Send(&pb.ListTasksResponse{
			Task:    task,
			Overdue: overdue,
		})
		return err
	})
}

func Filter(msg proto.Message, mask *fieldmaskpb.FieldMask) {
	if mask == nil || len(mask.Paths) == 0 {
		return
	}

	rft := msg.ProtoReflect()
	rft.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if !slices.Contains(mask.Paths, string(fd.Name())) {
			rft.Clear(fd)
		}
		return true
	})
}

func (s *server) UpdateTasks(stream pb.TodoService_UpdateTasksServer) error {
	for {
		req, err := stream.Recv()

		if err == io.EOF {
			return stream.SendAndClose(&pb.UpdateTasksResponse{})
		}
		if err != nil {
			return err
		}

		s.d.updateTask(
			req.Id,
			req.Description,
			req.DueDate.AsTime(),
			req.Done,
		)
	}
}

func (s *server) DeleteTasks(stream pb.TodoService_DeleteTasksServer) error {
	for {
		req, err := stream.Recv()

		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		s.d.deleteTask(req.Id)
		stream.Send(&pb.DeleteTasksResponse{})
	}
}
