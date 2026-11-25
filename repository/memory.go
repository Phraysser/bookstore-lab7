package repository

import (
	"context"
	"sync"

	"example.com/bookstore/book"
)

type Memory struct {
	mu    sync.RWMutex
	books map[string]book.Book
}

func NewMemory() *Memory {
	return &Memory{books: make(map[string]book.Book)}
}

func (r *Memory) Save(ctx context.Context, b book.Book) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}
	r.books[b.ID] = b
	return nil
}

func (r *Memory) Find(ctx context.Context, id string) (book.Book, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if err := ctx.Err(); err != nil {
		return book.Book{}, err
	}
	b, ok := r.books[id]
	if !ok {
		return book.Book{}, book.ErrNotFound
	}
	return b, nil
}

func (r *Memory) Update(ctx context.Context, id string, update func(*book.Book) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}
	b, ok := r.books[id]
	if !ok {
		return book.ErrNotFound
	}
	if err := update(&b); err != nil {
		return err
	}
	r.books[id] = b
	return nil
}
