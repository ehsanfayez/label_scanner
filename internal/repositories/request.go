package repositories

import (
	"context"
	"scanner/databases"

	"go.mongodb.org/mongo-driver/mongo"
)

type RequestRepo struct {
	collection *mongo.Collection
}

type Request struct {
	SerialNumbers []string `bson:"serial_number" json:"serial_number"`
	UUid          string   `bson:"uuid" json:"uuid"`
}

func NewRequestRepo() *RequestRepo {
	return &RequestRepo{
		collection: databases.DB.Collection("requests"),
	}
}

func (r *RequestRepo) Create(ctx context.Context, request *Request) (*Request, error) {
	_, err := r.collection.InsertOne(ctx, request)
	if err != nil {
		return nil, err
	}

	return request, nil
}

func (r *RequestRepo) FindByID(ctx context.Context, uuid string) (*Request, error) {
	request := &Request{}
	filter := map[string]interface{}{
		"uuid": uuid,
	}

	err := r.collection.FindOne(ctx, filter).Decode(request)
	if err != nil {
		return nil, err
	}

	return request, nil
}
