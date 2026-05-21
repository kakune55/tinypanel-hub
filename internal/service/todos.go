package service

import "tinypanel-hub/internal/domain"

type TodoService struct {
	store TodoStore
}

func (s TodoService) List() []domain.Todo {
	return s.store.Todos()
}

func (s TodoService) Get(id int64) (domain.Todo, bool) {
	return s.store.Todo(id)
}

func (s TodoService) Create(text string, status int) (domain.Todo, error) {
	return s.store.AddTodo(text, status)
}

func (s TodoService) Update(id, version int64, patch domain.TodoPatch) (domain.Todo, bool, bool, error) {
	return s.store.UpdateTodo(id, version, patch)
}

func (s TodoService) Delete(id, version int64) (bool, bool, error) {
	return s.store.DeleteTodo(id, version)
}
