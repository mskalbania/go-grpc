package db

import (
	"server/model"
	"time"
)

type TodoDB interface {
	AddTask(description string, dueDate time.Time) model.ID
	//GetTasks interface itself is not coupled to any iterator, loop, cursor implementation,
	// a user provided f function should be called on all tasks that were obtained
	GetTasks(applyOnEachRow func(item model.Task) error) error
}

func NewTodoDB() TodoDB {
	return &inMemoryTodoDB{data: make(map[model.ID]model.Task)}
}

type inMemoryTodoDB struct {
	data map[model.ID]model.Task
}

func (i *inMemoryTodoDB) AddTask(description string, dueDate time.Time) model.ID {
	id := model.ID(time.Now().UnixNano())
	i.data[id] = model.Task{
		ID:          id,
		Description: description,
		DueDate:     dueDate,
		Done:        false,
	}
	return id
}

// GetTasks now the implementation for the in memory db - we just call f on every item from the map,
// if error is received the iteration is stopped
func (i *inMemoryTodoDB) GetTasks(f func(item model.Task) error) error {
	for _, v := range i.data {
		if err := f(v); err != nil {
			return err
		}
	}
	return nil
}
