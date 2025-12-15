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
		SerialNumbers: []repositories.SerialCondition{},
		UUid:          uuid.New().String(),
	}

	for _, sn := range serialNumbers {
		request.SerialNumbers = append(request.SerialNumbers, repositories.SerialCondition{
			SerialNumber: sn,
			PsidStore:    false,
		})
	}

	return r.requestRepo.Create(ctx, request)
}

func (r *RequestService) GetRequestByID(ctx context.Context, uuid string) ([]string, error) {
	reques, err := r.requestRepo.FindByID(ctx, uuid)
	if err != nil {
		return nil, err
	}

	serials := []string{}
	for _, sc := range reques.SerialNumbers {
		if sc.PsidStore {
			continue
		}

		serials = append(serials, sc.SerialNumber)
	}

	return serials, nil
}

func (r *RequestService) UpdatePsidStore(ctx context.Context, uuid string, serialNumber string) error {
	return r.requestRepo.UpdatePsidStore(ctx, uuid, serialNumber)
}
