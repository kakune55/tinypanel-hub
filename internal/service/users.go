package service

import "tinypanel-hub/internal/domain"

type UserService struct {
	store UserStore
}

func (s UserService) ByTokenHash(tokenHash string) (domain.User, bool) {
	return s.store.UserByTokenHash(tokenHash)
}

func (s UserService) Get(id string) (domain.User, bool) {
	return s.store.User(id)
}

func (s UserService) Create(name, email, tokenHash string) (domain.User, error) {
	return s.store.CreateUser(name, email, tokenHash)
}
