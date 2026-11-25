package transport

import (
	"context"
	"errors"

	"example.com/bookstore/book"
	"example.com/bookstore/endpoint"
	"example.com/bookstore/pb"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type grpcServer struct {
	pb.UnimplementedBookServiceServer
	create, getByID, updateStock kitgrpc.Handler
}

func NewGRPC(ep endpoint.Endpoints) pb.BookServiceServer {
	return &grpcServer{
		create:      kitgrpc.NewServer(ep.Create, decodeGRPCCreate, encodeGRPCCreate),
		getByID:     kitgrpc.NewServer(ep.GetByID, decodeGRPCGet, encodeGRPCGet),
		updateStock: kitgrpc.NewServer(ep.UpdateStock, decodeGRPCUpdate, encodeGRPCUpdate),
	}
}

func (s *grpcServer) Create(ctx context.Context, req *pb.CreateRequest) (*pb.CreateResponse, error) {
	_, response, err := s.create.ServeGRPC(ctx, req)
	if err != nil {
		return nil, grpcError(err)
	}
	return response.(*pb.CreateResponse), nil
}

func (s *grpcServer) GetByID(ctx context.Context, req *pb.GetByIDRequest) (*pb.GetByIDResponse, error) {
	_, response, err := s.getByID.ServeGRPC(ctx, req)
	if err != nil {
		return nil, grpcError(err)
	}
	return response.(*pb.GetByIDResponse), nil
}

func (s *grpcServer) UpdateStock(ctx context.Context, req *pb.UpdateStockRequest) (*pb.UpdateStockResponse, error) {
	_, response, err := s.updateStock.ServeGRPC(ctx, req)
	if err != nil {
		return nil, grpcError(err)
	}
	return response.(*pb.UpdateStockResponse), nil
}

func decodeGRPCCreate(_ context.Context, request any) (any, error) {
	req := request.(*pb.CreateRequest)
	stock := int(req.GetStock())
	if int64(stock) != req.GetStock() {
		return nil, status.Error(codes.InvalidArgument, "stock exceeds the supported range")
	}
	return book.Book{Title: req.GetTitle(), Author: req.GetAuthor(), Price: req.GetPrice(), Stock: stock}, nil
}

func decodeGRPCGet(_ context.Context, request any) (any, error) {
	return request.(*pb.GetByIDRequest).GetId(), nil
}

func decodeGRPCUpdate(_ context.Context, request any) (any, error) {
	req := request.(*pb.UpdateStockRequest)
	delta := int(req.GetDelta())
	if int64(delta) != req.GetDelta() {
		return nil, status.Error(codes.InvalidArgument, "delta exceeds the supported range")
	}
	return endpoint.UpdateStockRequest{ID: req.GetId(), Delta: delta}, nil
}

func encodeGRPCCreate(_ context.Context, response any) (any, error) {
	return &pb.CreateResponse{Id: response.(endpoint.CreateResponse).ID}, nil
}

func encodeGRPCGet(_ context.Context, response any) (any, error) {
	b := response.(book.Book)
	return &pb.GetByIDResponse{
		Book: &pb.Book{Id: b.ID, Title: b.Title, Author: b.Author, Price: b.Price, Stock: int64(b.Stock), InStock: b.InStock},
	}, nil
}

func encodeGRPCUpdate(context.Context, any) (any, error) {
	return &pb.UpdateStockResponse{}, nil
}

func grpcError(err error) error {
	if _, ok := status.FromError(err); ok {
		return err
	}
	switch {
	case errors.Is(err, book.ErrInvalidBook), errors.Is(err, book.ErrInvalidID):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, book.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, book.ErrInvalidStock):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return status.FromContextError(err).Err()
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
