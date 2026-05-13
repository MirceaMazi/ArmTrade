package db

import (
	"context"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	Client *mongo.Client
	DB     *mongo.Database
)

func InitDatabase() {
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}

	dbName := os.Getenv("MONGO_DB")
	if dbName == "" {
		dbName = "armtrade"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	Client, err = mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	if err := Client.Ping(ctx, nil); err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}

	DB = Client.Database(dbName)

	// Create indexes
	usersCol := DB.Collection("users")
	_, err = usersCol.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "username", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		log.Fatalf("Failed to create users index: %v", err)
	}

	watchlistCol := DB.Collection("watchlist")
	// Drop old unique index if it exists (we now allow multiple positions per ticker)
	_ = watchlistCol.Indexes().DropOne(ctx, "user_id_1_ticker_1")
	_, err = watchlistCol.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "user_id", Value: 1},
			{Key: "ticker", Value: 1},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create watchlist index: %v", err)
	}

	log.Printf("MongoDB connected successfully (%s/%s)\n", uri, dbName)
}

func Disconnect() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return Client.Disconnect(ctx)
}

func Users() *mongo.Collection {
	return DB.Collection("users")
}

func Watchlist() *mongo.Collection {
	return DB.Collection("watchlist")
}

func Analyses() *mongo.Collection {
	return DB.Collection("analyses")
}
