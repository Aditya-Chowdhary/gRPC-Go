package main

import (
	"time"

	pb "github.com/Aditya-Chowdhary/gRPC-Go/proto/todo/v2"
)

type server struct {
	d db
	pb.UnimplementedTodoServiceServer
}

type db interface {
	addTask(description string, dueDate time.Time) (uint64, error)
	getTasks(f func(interface{}) error) error
	updateTask(id uint64, description string, dueDate time.Time, done bool) error
	deleteTask(id uint64) error
}
