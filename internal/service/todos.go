package service

import "tinypanel-hub/internal/domain"

type TodoService struct {
	store TodoStore
}

func (s TodoService) List(ownerID string) []domain.Todo {
	return s.store.Todos(ownerID)
}

func (s TodoService) Get(ownerID string, id int64) (domain.Todo, bool) {
	return s.store.Todo(ownerID, id)
}

func (s TodoService) Create(ownerID, text string, status int) (domain.Todo, error) {
	return s.store.AddTodo(ownerID, text, status)
}

func (s TodoService) Update(ownerID string, id, version int64, patch domain.TodoPatch) (domain.Todo, bool, bool, error) {
	return s.store.UpdateTodo(ownerID, id, version, patch)
}

func (s TodoService) Delete(ownerID string, id, version int64) (bool, bool, error) {
	return s.store.DeleteTodo(ownerID, id, version)
}
