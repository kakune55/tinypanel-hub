package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"tinypanel-hub/internal/domain"
)

type stateFile struct {
	path string
	data stateData
}

type stateData struct {
	NextMessageID int64            `json:"next_message_id"`
	NextTodoID    int64            `json:"next_todo_id"`
	Weather       domain.Weather   `json:"weather"`
	Users         []domain.User    `json:"users"`
	Devices       []domain.Device  `json:"devices"`
	Messages      []domain.Message `json:"messages"`
	Todos         []domain.Todo    `json:"todos"`
}

func newStateFile(path string) *stateFile {
	return &stateFile{
		path: path,
		data: stateData{
			NextMessageID: 1,
			NextTodoID:    1,
			Weather:       defaultWeather(),
		},
	}
}

func (f *stateFile) load() error {
	b, err := os.ReadFile(f.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(b) == 0 {
		return nil
	}

	if err := json.Unmarshal(b, &f.data); err != nil {
		return err
	}
	f.normalize()
	return nil
}

func (f *stateFile) save() error {
	if err := os.MkdirAll(filepath.Dir(f.path), 0755); err != nil {
		return err
	}

	b, err := json.MarshalIndent(f.data, "", "  ")
	if err != nil {
		return err
	}

	tmp := f.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, f.path)
}

func (f *stateFile) normalize() {
	if f.data.NextMessageID == 0 {
		f.data.NextMessageID = int64(len(f.data.Messages)) + 1
	}
	if f.data.NextTodoID == 0 {
		f.data.NextTodoID = nextTodoID(f.data.Todos)
	}
	if f.data.Weather.Location == "" {
		f.data.Weather = defaultWeather()
	}
	for i := range f.data.Messages {
		if f.data.Messages[i].Priority == "" {
			f.data.Messages[i].Priority = domain.MessagePriorityNormal
		}
		if f.data.Messages[i].Status == "" {
			f.data.Messages[i].Status = domain.MessageStatusPending
		}
	}
}
