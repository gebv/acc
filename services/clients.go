package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"time"
)

//go:generate reform

//reform:acca.clients
type Client struct {
	ClientID    int64     `reform:"client_id,pk"`
	AccessToken string    `reform:"access_token"`
	CreatedAt   time.Time `reform:"created_at"`
}

func (d *Client) BeforeInsert() error {
	d.CreatedAt = time.Now()
	return nil
}

// GetClient возвращает данные клиента из контекста.
//
// Публичный метод для использования в других пакетах
func GetClient(ctx context.Context) *Client {
	return ctx.Value(clientCtxKey).(*Client)
}

func NewClient() *Client {
	return &Client{
		AccessToken: randString(32),
	}
}

func randString(len int) string {
	b := make([]byte, len)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
