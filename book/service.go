package book

import (
	"context"
	"errors"
	"math"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrInvalidBook  = errors.New("title and author are required, price and stock must be nonnegative")
	ErrInvalidID    = errors.New("book ID is required")
	ErrNotFound     = errors.New("book not found")
	ErrInvalidStock = errors.New("resulting stock is outside the supported range")
)

type Book struct {
	ID      string  `json:"id"`
	Title   string  `json:"title"`
	Author  string  `json:"author"`
	Price   float64 `json:"price"`
	Stock   int     `json:"stock"`
	InStock bool    `json:"in_stock"`
}

type Service interface {
	Create(ctx context.Context, book Book) (string, error)
	GetByID(ctx context.Context, id string) (Book, error)
	UpdateStock(ctx context.Context, id string, delta int) error
}

type Repository interface {
	Save(context.Context, Book) error
	Find(context.Context, string) (Book, error)
	Update(context.Context, string, func(*Book) error) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, b Book) (string, error) {
	b.Title = strings.TrimSpace(b.Title)
	b.Author = strings.TrimSpace(b.Author)
	if b.Title == "" || b.Author == "" || b.Stock < 0 || b.Price < 0 || math.IsNaN(b.Price) || math.IsInf(b.Price, 0) {
		return "", ErrInvalidBook
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	b.ID = id.String()
	b.InStock = b.Stock > 0
	if err := s.repo.Save(ctx, b); err != nil {
		return "", err
	}
	return b.ID, nil
}

func (s *service) GetByID(ctx context.Context, id string) (Book, error) {
	if strings.TrimSpace(id) == "" {
		return Book{}, ErrInvalidID
	}
	return s.repo.Find(ctx, id)
}

func (s *service) UpdateStock(ctx context.Context, id string, delta int) error {
	if strings.TrimSpace(id) == "" {
		return ErrInvalidID
	}
	return s.repo.Update(ctx, id, func(b *Book) error {
		if delta < -b.Stock || delta > math.MaxInt-b.Stock {
			return ErrInvalidStock
		}
		b.Stock += delta
		b.InStock = b.Stock > 0
		return nil
	})
}
