package store

import (
	"time"

	"tinypanel-hub/internal/domain"
)

func (s *FileStore) Todos(ownerID string) []domain.Todo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.ownerTodosLocked(ownerID)
}

func (s *FileStore) Todo(ownerID string, id int64) (domain.Todo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	index := s.todoIndex(ownerID, id)
	if index < 0 {
		return domain.Todo{}, false
	}
	return s.state.data.Todos[index], true
}

func (s *FileStore) AddTodo(ownerID, text string, status int) (domain.Todo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	todo := domain.Todo{
		ID:        s.state.data.NextTodoID,
		OwnerID:   ownerID,
		Text:      text,
		Status:    status,
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.state.data.NextTodoID++
	s.state.data.Todos = append(s.state.data.Todos, todo)
	return todo, s.state.save()
}

func (s *FileStore) UpdateTodo(ownerID string, id, version int64, patch domain.TodoPatch) (domain.Todo, bool, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.todoIndex(ownerID, id)
	if index < 0 {
		return domain.Todo{}, false, false, nil
	}

	todo := s.state.data.Todos[index]
	if todo.Version != version {
		return todo, true, false, nil
	}
	if patch.Text != nil {
		todo.Text = *patch.Text
	}
	if patch.Status != nil {
		todo.Status = *patch.Status
	}
	todo.Version++
	todo.UpdatedAt = time.Now().UTC()
	s.state.data.Todos[index] = todo
	return todo, true, true, s.state.save()
}

func (s *FileStore) DeleteTodo(ownerID string, id, version int64) (bool, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.todoIndex(ownerID, id)
	if index < 0 {
		return false, false, nil
	}
	if s.state.data.Todos[index].Version != version {
		return true, false, nil
	}

	s.state.data.Todos = append(s.state.data.Todos[:index], s.state.data.Todos[index+1:]...)
	return true, true, s.state.save()
}

func (s *FileStore) todoIndex(ownerID string, id int64) int {
	for i, todo := range s.state.data.Todos {
		if todo.ID == id && todo.OwnerID == ownerID {
			return i
		}
	}
	return -1
}

func (s *FileStore) ownerTodosLocked(ownerID string) []domain.Todo {
	out := []domain.Todo{}
	for _, todo := range s.state.data.Todos {
		if todo.OwnerID == ownerID {
			out = append(out, todo)
		}
	}
	return out
}

func nextTodoID(items []domain.Todo) int64 {
	next := int64(1)
	for _, item := range items {
		if item.ID >= next {
			next = item.ID + 1
		}
	}
	return next
}
