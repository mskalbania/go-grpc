package model

import (
	"google.golang.org/protobuf/types/known/timestamppb"
	"proto/gen/todo"
	"time"
)

type ID uint64

type Task struct {
	ID          ID
	Description string
	Done        bool
	DueDate     time.Time
}

func (t Task) ToProto() *todo.Task {
	return &todo.Task{
		Id:          uint64(t.ID),
		Description: t.Description,
		Done:        t.Done,
		DueDate:     timestamppb.New(t.DueDate),
	}
}
