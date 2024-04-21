package main

import (
	"context"

	pb "github.com/Aditya-Chowdhary/gRPC-Go/proto/todo/v1"
)

func (s *server) AddTask(_ context.Context, in *pb.AddTaskRequest) (*pb.AddTaskResponse, error) {
	id, _ := s.d.addTask(in.Description, in.DueDate.AsTime())

	return &pb.AddTaskResponse{Id: id}, nil
}