package endpoint

import (
	"context"

	"example.com/bookstore/book"
	kit "github.com/go-kit/kit/endpoint"
)

type Endpoints struct {
	Create      kit.Endpoint
	GetByID     kit.Endpoint
	UpdateStock kit.Endpoint
}

type CreateResponse struct {
	ID string `json:"id"`
}

type UpdateStockRequest struct {
	ID    string
	Delta int
}

func New(s book.Service) Endpoints {
	return Endpoints{
		Create: func(ctx context.Context, request any) (any, error) {
			id, err := s.Create(ctx, request.(book.Book))
			return CreateResponse{ID: id}, err
		},
		GetByID: func(ctx context.Context, request any) (any, error) {
			return s.GetByID(ctx, request.(string))
		},
		UpdateStock: func(ctx context.Context, request any) (any, error) {
			req := request.(UpdateStockRequest)
			return nil, s.UpdateStock(ctx, req.ID, req.Delta)
		},
	}
}
