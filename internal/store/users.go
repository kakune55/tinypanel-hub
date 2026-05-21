package store

import (
	"strings"
	"time"

	"tinypanel-hub/internal/domain"
)

func (s *FileStore) UserByTokenHash(tokenHash string) (domain.User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, user := range s.state.data.Users {
		if user.APITokenHash == tokenHash {
			return user, true
		}
	}
	return domain.User{}, false
}

func (s *FileStore) User(id string) (domain.User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, user := range s.state.data.Users {
		if user.ID == id {
			return user, true
		}
	}
	return domain.User{}, false
}

func (s *FileStore) CreateUser(name, email, tokenHash string) (domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user := domain.User{
		ID:           "usr_" + randomHex(8),
		Name:         strings.TrimSpace(name),
		Email:        strings.TrimSpace(email),
		APITokenHash: tokenHash,
		CreatedAt:    time.Now().UTC(),
	}
	if user.Name == "" {
		user.Name = "user"
	}
	s.state.data.Users = append(s.state.data.Users, user)
	return user, s.state.save()
}
