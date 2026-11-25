package book_test

import (
	"context"
	"errors"
	"testing"

	"example.com/bookstore/book"
	"example.com/bookstore/repository"
)

func TestStockLifecycle(t *testing.T) {
	ctx := context.Background()
	svc := book.NewService(repository.NewMemory())
	id, err := svc.Create(ctx, book.Book{Title: "Go in Action", Author: "William Kennedy", Price: 39.90, Stock: 3})
	if err != nil || id == "" {
		t.Fatalf("Create = %q, %v, want a nonempty ID", id, err)
	}
	checkStock := func(want int) {
		t.Helper()
		result, err := svc.GetByID(ctx, id)
		if err != nil || result.ID != id || result.Stock != want || result.InStock != (want > 0) {
			t.Fatalf("GetByID = %+v, %v, want stock %d", result, err, want)
		}
	}
	checkStock(3)
	if err := svc.UpdateStock(ctx, id, -3); err != nil {
		t.Fatal(err)
	}
	checkStock(0)
	if err := svc.UpdateStock(ctx, id, -1); !errors.Is(err, book.ErrInvalidStock) {
		t.Fatalf("UpdateStock = %v, want ErrInvalidStock", err)
	}
	checkStock(0)
}

func TestInvalidBookAndMissingID(t *testing.T) {
	ctx := context.Background()
	svc := book.NewService(repository.NewMemory())
	if _, err := svc.Create(ctx, book.Book{Title: "Book", Author: "Author", Price: -1}); !errors.Is(err, book.ErrInvalidBook) {
		t.Fatalf("Create = %v, want ErrInvalidBook", err)
	}
	if _, err := svc.GetByID(ctx, "missing"); !errors.Is(err, book.ErrNotFound) {
		t.Fatalf("GetByID = %v, want ErrNotFound", err)
	}
}
