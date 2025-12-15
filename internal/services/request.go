package services

import (
	"context"
	"scanner/internal/repositories"

	"github.com/google/uuid"
)

type RequestService struct {
	requestRepo *repositories.RequestRepo
}

func NewRequestService() *RequestService {
	return &RequestService{
		requestRepo: repositories.NewRequestRepo(),
	}
}

func (r *RequestService) CreateRequest(ctx context.Context, serialNumbers []string) (*repositories.Request, error) {
	request := &repositories.Request{
		SerialNumbers: serialNumbers,
		UUid:          uuid.New().String(),
	}

	return r.requestRepo.Create(ctx, request)
}

func (r *RequestService) GetRequestByID(ctx context.Context, uuid string) (*repositories.Request, error) {
	return r.requestRepo.FindByID(ctx, uuid)
}
