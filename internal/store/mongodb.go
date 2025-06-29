package store

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	client   *mongo.Client
	database *mongo.Database
)

// Connect initializes the MongoDB connection.
// It should be called once at application startup.
func Connect(uri, dbName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	client, err = mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return err
	}

	// Ping the primary
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return err
	}

	database = client.Database(dbName)
	log.Println("Successfully connected to MongoDB:", dbName)
	return nil
}

// Disconnect closes the MongoDB connection.
// It should be called once on application shutdown.
func Disconnect() {
	if client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := client.Disconnect(ctx); err != nil {
			log.Printf("Error disconnecting from MongoDB: %v", err)
		} else {
			log.Println("Disconnected from MongoDB.")
		}
	}
}

// GetDB returns the MongoDB database instance.
// It should only be called after a successful Connect.
func GetDB() *mongo.Database {
	if database == nil {
		// This should not happen if Connect is called at startup
		log.Fatal("MongoDB database instance is not initialized. Call Connect first.")
	}
	return database
}

// GetClient returns the MongoDB client instance.
// It should only be called after a successful Connect.
func GetClient() *mongo.Client {
	if client == nil {
		// This should not happen if Connect is called at startup
		log.Fatal("MongoDB client instance is not initialized. Call Connect first.")
	}
	return client
}
