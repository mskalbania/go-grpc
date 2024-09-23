package api

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
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

func (t *TodoAPI) ListTasks(rq *todo.ListTasksRequest, server grpc.ServerStreamingServer[todo.ListTasksResponse]) error {
	err := t.db.GetTasks(func(task model.Task) error {
		overdue := task.Done == false && time.Now().UTC().After(task.DueDate)
		if err := server.Send(&todo.ListTasksResponse{
			Task:    filter(rq.GetMask(), task).ToProto(),
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

func filter(mask *fieldmaskpb.FieldMask, task model.Task) model.Task {
	if mask == nil {
		return task
	}
	maskedTask := model.Task{}
	for _, path := range mask.Paths {
		switch path {
		case "id":
			maskedTask.ID = task.ID
		case "description":
			maskedTask.Description = task.Description
		}
	}
	return maskedTask
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
		err = t.db.UpdateTask(model.ID(rq.Id), rq.Description, rq.DueDate.AsTime(), rq.Done)
		if err != nil {
			log.Printf("error updating task: %v", err)
			return status.Error(codes.Internal, "error updating task")
		}
		log.Printf("updated task with id: %d, description: %s, dueDate: %s, done: %v", rq.Id, rq.Description, rq.DueDate.AsTime().String(), rq.Done)
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
