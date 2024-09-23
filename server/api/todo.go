package api

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
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
		return nil
	})
	if err != nil {
		log.Printf("error getting tasks: %v", err)
		return status.Error(codes.Internal, "error getting tasks")
	}
	return nil
}

func (t *TodoAPI) UpdateTask(server grpc.ClientStreamingServer[todo.UpdateTaskRequest, todo.UpdateTaskResponse]) error {
	for { //now inf loop on server end to consume the client stream
		rq, err := server.Recv()
		if err == io.EOF { //until client sends half close
			log.Println("client done")
			return server.SendAndClose(&todo.UpdateTaskResponse{}) //once client done send rs and trailer
		}
		if err != nil {
			log.Printf("stream closed unexpectedly: %v", err)
			return err
		}
		err = t.db.UpdateTask(model.ID(rq.Task.Id), rq.Task.Description, rq.Task.DueDate.AsTime(), rq.Task.Done)
		if err != nil {
			log.Printf("error updating task: %v", err)
			return status.Error(codes.Internal, "error updating task")
		}
		log.Printf("updated task with id: %d, description: %s, dueDate: %s, done: %v", rq.Task.Id, rq.Task.Description, rq.Task.DueDate.AsTime().String(), rq.Task.Done)
	}
}

func (t *TodoAPI) DeleteTask(server grpc.BidiStreamingServer[todo.DeleteTaskRequest, todo.DeleteTaskResponse]) error {
	for {
		rq, err := server.Recv()
		if err == io.EOF {
			log.Println("client done")
			return nil
		}
		if err != nil {
			log.Printf("stream closed unexpectedly: %v", err)
			return err
		}
		err = t.db.DeleteTask(model.ID(rq.Id))
		if err != nil {
			log.Printf("error deleting task: %v", err)
			return status.Error(codes.Internal, "error deleting task")
		}
		log.Printf("deleted task with id: %d", rq.Id)
		err = server.Send(&todo.DeleteTaskResponse{})
		if err != nil {
			return err
		}
	}
}
