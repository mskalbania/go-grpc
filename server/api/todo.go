package api

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"proto/gen/todo"
	"server/db"
	"server/model"
	"time"
)

type TodoAPI struct {
	todo.UnimplementedTodoServiceServer
	db db.TodoDB
}

func NewTodoAPI(db db.TodoDB) *TodoAPI {
	return &TodoAPI{db: db}
}

func (t *TodoAPI) AddTask(_ context.Context, rq *todo.AddTaskRequest) (*todo.AddTaskResponse, error) {
	id := t.db.AddTask(rq.Description, rq.DueDate.AsTime())
	log.Printf("added task with: id: %d, description: %s, dueDate: %s", id, rq.Description, rq.DueDate.AsTime().String())
	return &todo.AddTaskResponse{
		Id: uint64(id),
	}, nil
}

func (t *TodoAPI) ListTasks(_ *todo.ListTasksRequest, server grpc.ServerStreamingServer[todo.ListTasksResponse]) error {
	err := t.db.GetTasks(func(task model.Task) error {
		overdue := task.Done == false && time.Now().UTC().After(task.DueDate)
		if err := server.Send(&todo.ListTasksResponse{
			Task:    task.ToProto(),
			Overdue: overdue,
		}); err != nil {
			return err
		}
		time.Sleep(time.Second) //simulate some processing
		return nil
	})
	if err != nil {
		log.Printf("error getting tasks: %v", err)
		return status.Error(codes.Internal, "error getting tasks")
	}
	return nil
}
