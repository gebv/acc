package services

import "gopkg.in/reform.v1"

func NewService(db *reform.DB) *Service {
	return &Service{
		db: db,
	}
}

type Service struct {
	db *reform.DB
}
