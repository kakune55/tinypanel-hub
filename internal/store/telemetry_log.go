package store

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	"tinypanel-hub/internal/domain"
)

type telemetryLog struct {
	path string
}

func newTelemetryLog(path string) *telemetryLog {
	return &telemetryLog{path: path}
}

func (l *telemetryLog) loadRecent(limit int) ([]domain.Telemetry, error) {
	f, err := os.Open(l.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var items []domain.Telemetry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var item domain.Telemetry
		if err := json.Unmarshal(line, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
		if limit > 0 && len(items) > limit {
			copy(items, items[len(items)-limit:])
			items = items[:limit]
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (l *telemetryLog) append(item domain.Telemetry) error {
	if err := os.MkdirAll(filepath.Dir(l.path), 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	return writeJSONLine(f, item)
}

func writeJSONLine(w io.Writer, value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if _, err := w.Write(b); err != nil {
		return err
	}
	_, err = w.Write([]byte("\n"))
	return err
}
