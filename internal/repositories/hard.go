package repositories

import (
	"context"
	"scanner/databases"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var MainKeys = []string{
	"capacity",
	"hard_type",
	"make",
	"model",
	"part_number",
	"serial_number",
	"psid",
	"inventory_id",
	"eui",
}

type Hard struct {
	ID           primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	Capacity     string                 `bson:"capacity" json:"capacity"`
	Eui          string                 `bson:"eui" json:"eui"`
	Type         string                 `bson:"type" json:"hard_type"`
	InventoryID  string                 `bson:"inventory_id" json:"inventory_id"`
	Make         string                 `bson:"make" json:"make"`
	Model        string                 `bson:"model" json:"model"`
	PartNumber   string                 `bson:"part_number" json:"part_number"`
	SerialNumber string                 `bson:"serial_number" json:"serial_number"`
	Psid         string                 `bson:"psid" json:"psid"`
	ExtraFileds  map[string]interface{} `bson:"extra_fields" json:"extra_fields"`
	Images       []string               `bson:"images,omitempty" json:"images,omitempty"`
}

type HardRepository struct {
	collection *mongo.Collection
}

func NewHardRepository() *HardRepository {
	return &HardRepository{
		collection: databases.DB.Collection("hards"),
	}
}

type HardFilter struct {
	SerialNumber string `json:"serial_number" form:"serial_number"`
	Make         string `json:"make" form:"make"`
	// InventoryID  string `json:"inventory_id" form:"inventory_id"`
}

func (r *HardRepository) FindByInput(ctx context.Context, data HardFilter) (*Hard, error) {
	var hard Hard
	filer := make(map[string]interface{})
	if data.SerialNumber != "" {
		filer["serial_number"] = data.SerialNumber
	}

	if data.Make != "" {
		filer["make"] = data.Make
	}

	err := r.collection.FindOne(ctx, filer).Decode(&hard)
	if err != nil {
		return nil, err
	}

	// if data.InventoryID != "" {
	// 	err := r.collection.FindOne(ctx, map[string]interface{}{
	// 		"inventory_id": data.InventoryID,
	// 	}).Decode(&hard)

	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }

	return &hard, nil
}

func (r *HardRepository) Insert(ctx context.Context, hard *Hard) error {
	_, err := r.collection.InsertOne(ctx, hard)
	return err
}

func (r *HardRepository) Update(ctx context.Context, id string, hard *Hard) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	update := map[string]interface{}{
		"$set": hard,
	}

	_, err = r.collection.UpdateByID(ctx, objID, update)
	return err
}
