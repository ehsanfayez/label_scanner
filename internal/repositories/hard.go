package repositories

import (
	"context"
	"fmt"
	"scanner/databases"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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
	ID            primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	Capacity      string                 `bson:"capacity" json:"capacity"`
	Eui           string                 `bson:"eui" json:"eui"`
	Type          string                 `bson:"type" json:"hard_type"`
	InventoryID   string                 `bson:"inventory_id" json:"inventory_id"`
	Make          string                 `bson:"make" json:"make"`
	Model         string                 `bson:"model" json:"model"`
	PartNumber    string                 `bson:"part_number" json:"part_number"`
	SerialNumber  string                 `bson:"serial_number" json:"serial_number"`
	Psid          string                 `bson:"psid" json:"psid"`
	ExtraFileds   map[string]interface{} `bson:"extra_fields" json:"extra_fields"`
	Images        []string               `bson:"images" json:"images"`
	WipeAccepted  bool                   `bson:"vipe_accepted" json:"wipe_accepted"`
	UserEdited    bool                   `bson:"user_edited" json:"user_edited"`
	IncorrectPsid bool                   `bson:"incorrect_psid" json:"-"`
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
	InventoryID  string `json:"inventory_id" form:"inventory_id"`
}

type AddHardFilter struct {
	SerialNumber string `json:"serial_number" form:"serial_number"`
	Psid         string `json:"psid" form:"psid"`
}

func (r *HardRepository) FindByInput(ctx context.Context, data *HardFilter) ([]Hard, error) {
	hards := []Hard{}
	filer := make(map[string]interface{})
	if data.SerialNumber != "" {
		filer["serial_number"] = data.SerialNumber
	}

	if data.Make != "" {
		filer["make"] = data.Make
	}

	if data.InventoryID != "" {
		filer["inventory_id"] = data.InventoryID
	}

	if data.Make == "" && data.SerialNumber == "" {
		return nil, fmt.Errorf("serial number must be provided")
	}

	// Filter out records with incorrect_psid = false
	filer["incorrect_psid"] = bson.M{"$ne": true}

	// each record has vipe_accepted = true shoud be upper then records with user_edited = true then other records
	findOptions := options.Find().SetSort(bson.D{
		{Key: "vipe_accepted", Value: -1},
		{Key: "user_edited", Value: -1},
	})

	cursor, err := r.collection.Find(ctx, filer, findOptions)
	if err != nil {
		return nil, err
	}

	err = cursor.All(ctx, &hards)
	if err != nil {
		return nil, err
	}

	return hards, nil
}

func (r *HardRepository) FindByID(ctx context.Context, id string) (*Hard, error) {
	hard := &Hard{}
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	err = r.collection.FindOne(ctx, map[string]interface{}{
		"_id": objID,
	}).Decode(&hard)

	if err != nil {
		return nil, err
	}

	return hard, nil
}

func (r *HardRepository) FindByPsid(ctx context.Context, data AddHardFilter) (*Hard, error) {
	hard := &Hard{}
	filter := make(map[string]interface{})
	if data.Psid != "" {
		filter["psid"] = data.Psid
	}

	if data.SerialNumber != "" {
		filter["serial_number"] = data.SerialNumber
	}

	if data.Psid == "" && data.SerialNumber == "" {
		return nil, fmt.Errorf("psid or serial number must be provided")
	}

	err := r.collection.FindOne(ctx, filter).Decode(&hard)

	if err != nil {
		return nil, err
	}

	return hard, nil
}

func (r *HardRepository) WipeAccepted(ctx context.Context, hard *Hard) error {
	update := map[string]interface{}{
		"$set": map[string]interface{}{
			"vipe_accepted": true,
		},
	}

	_, err := r.collection.UpdateOne(ctx, map[string]interface{}{
		"_id": hard.ID,
	}, update)

	if err != nil {
		return err
	}

	return nil
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

func (r *HardRepository) DeleteByPsid(ctx context.Context, hard *Hard) error {
	update := map[string]interface{}{
		"$set": map[string]interface{}{
			"incorrect_psid": true,
		},
	}

	_, err := r.collection.UpdateOne(ctx, map[string]interface{}{
		"_id": hard.ID,
	}, update)

	if err != nil {
		return err
	}

	return nil
}
