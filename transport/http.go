package transport

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"example.com/bookstore/book"
	"example.com/bookstore/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
)

func NewHTTP(ep endpoint.Endpoints) http.Handler {
	options := kithttp.ServerErrorEncoder(encodeHTTPError)
	mux := http.NewServeMux()
	mux.Handle("POST /books", kithttp.NewServer(ep.Create, decodeHTTPCreate, encodeHTTPCreated, options))
	mux.Handle("GET /books/{id}", kithttp.NewServer(ep.GetByID, decodeHTTPGet, kithttp.EncodeJSONResponse, options))
	mux.Handle("PATCH /books/{id}/stock", kithttp.NewServer(ep.UpdateStock, decodeHTTPUpdate, encodeHTTPNoContent, options))
	return mux
}

var errInvalidJSON = errors.New("invalid JSON request")

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, (1<<20)+1))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return errInvalidJSON
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return errInvalidJSON
	}
	return nil
}

func decodeHTTPCreate(_ context.Context, r *http.Request) (any, error) {
	var input struct {
		Title  string  `json:"title"`
		Author string  `json:"author"`
		Price  float64 `json:"price"`
		Stock  int     `json:"stock"`
	}
	if err := decodeJSON(r, &input); err != nil {
		return nil, err
	}
	return book.Book{Title: input.Title, Author: input.Author, Price: input.Price, Stock: input.Stock}, nil
}

func decodeHTTPGet(_ context.Context, r *http.Request) (any, error) {
	return r.PathValue("id"), nil
}

func decodeHTTPUpdate(_ context.Context, r *http.Request) (any, error) {
	var input struct {
		Delta *int `json:"delta"`
	}
	if err := decodeJSON(r, &input); err != nil {
		return nil, err
	}
	if input.Delta == nil {
		return nil, errInvalidJSON
	}
	return endpoint.UpdateStockRequest{ID: r.PathValue("id"), Delta: *input.Delta}, nil
}

func encodeHTTPCreated(_ context.Context, w http.ResponseWriter, response any) error {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", "/books/"+response.(endpoint.CreateResponse).ID)
	w.WriteHeader(http.StatusCreated)
	return json.NewEncoder(w).Encode(response)
}

func encodeHTTPNoContent(_ context.Context, w http.ResponseWriter, _ any) error {
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func encodeHTTPError(_ context.Context, err error, w http.ResponseWriter) {
	code, message := http.StatusInternalServerError, "internal server error"
	switch {
	case errors.Is(err, errInvalidJSON), errors.Is(err, book.ErrInvalidBook), errors.Is(err, book.ErrInvalidID):
		code, message = http.StatusBadRequest, err.Error()
	case errors.Is(err, book.ErrNotFound):
		code, message = http.StatusNotFound, err.Error()
	case errors.Is(err, book.ErrInvalidStock):
		code, message = http.StatusConflict, err.Error()
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
