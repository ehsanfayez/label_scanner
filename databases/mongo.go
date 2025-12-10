package databases

import (
	"context"
	"log"
	"scanner/config"

	"fmt"

	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	mongoOnce   sync.Once
	mongoClient *mongo.Client
	DB          *mongo.Database
)

func InitialMongoDB(config *config.Config) {
	mongoOnce.Do(func() {
		uri := fmt.Sprintf("mongodb://%s:%s@%s:%d",
			config.MongoDB.Username,
			config.MongoDB.Password,
			config.MongoDB.Host,
			config.MongoDB.Port)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var err error
		mongoClient, err = mongo.Connect(ctx, options.Client().ApplyURI(uri).SetMaxPoolSize(10))
		if err != nil {
			log.Fatalf("Failed to connect to MongoDB: %v", err.Error())
		}

		if err = mongoClient.Ping(ctx, nil); err != nil {
			log.Fatalf("MongoDB unreachable: %v", err.Error())
		}

		DB = mongoClient.Database(config.MongoDB.Database)
	})
}

func CloseMongoDB() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := mongoClient.Disconnect(ctx); err != nil {
		log.Fatalf("Failed to disconnect from MongoDB: %v", err.Error())
	}
}
