package api

import "proto/gen/todo"

type TodoAPI struct {
	todo.UnimplementedTodoServer
}

func NewTodoAPI() *TodoAPI {
	return &TodoAPI{}
}
