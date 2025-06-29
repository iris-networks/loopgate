package store

import (
	"context"
	"errors" // Added import for errors package
	"log"
	"loopgate/internal/types"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	sessionsCollectionName    = "sessions"
	hitlRequestsCollectionName = "hitl_requests"
)

// EnsureIndexes creates necessary indexes for the collections if they don't already exist.
func EnsureIndexes(db *mongo.Database) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Sessions collection indexes
	sessionsCollection := db.Collection(sessionsCollectionName)
	sessionIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "client_id", Value: 1}},
			Options: options.Index().SetUnique(false), // client_id might not be unique if sessions can be re-registered
		},
		{
			Keys:    bson.D{{Key: "active", Value: 1}},
			Options: options.Index(),
		},
	}
	_, err := sessionsCollection.Indexes().CreateMany(ctx, sessionIndexes)
	if err != nil {
		return err
	}
	log.Println("Session indexes ensured.")

	// HITLRequests collection indexes
	hitlRequestsCollection := db.Collection(hitlRequestsCollectionName)
	requestIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "session_id", Value: 1}},
			Options: options.Index(),
		},
		{
			Keys:    bson.D{{Key: "client_id", Value: 1}},
			Options: options.Index(),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index(),
		},
		{
			Keys:    bson.D{{Key: "created_at", Value: -1}}, // For sorting by newest
			Options: options.Index(),
		},
	}
	_, err = hitlRequestsCollection.Indexes().CreateMany(ctx, requestIndexes)
	if err != nil {
		return err
	}
	log.Println("HITL request indexes ensured.")

	return nil
}

// MongoStoreRequest stores a new HITL request in MongoDB.
func MongoStoreRequest(db *mongo.Database, request *types.HITLRequest) error {
	collection := db.Collection(hitlRequestsCollectionName)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := collection.InsertOne(ctx, request)
	return err
}

// MongoGetRequest retrieves a HITL request by its ID from MongoDB.
func MongoGetRequest(db *mongo.Database, requestID string) (*types.HITLRequest, error) {
	collection := db.Collection(hitlRequestsCollectionName)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var request types.HITLRequest
	err := collection.FindOne(ctx, bson.M{"_id": requestID}).Decode(&request)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, mongo.ErrNoDocuments // Return mongo.ErrNoDocuments
		}
		return nil, err
	}
	return &request, nil
}

// MongoUpdateRequestResponse updates a HITL request with the human's response.
func MongoUpdateRequestResponse(db *mongo.Database, requestID string, humanResponse string, approved bool, status types.RequestStatus, respondedAt time.Time) error {
	collection := db.Collection(hitlRequestsCollectionName)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": requestID}
	update := bson.M{
		"$set": bson.M{
			"response":    humanResponse,
			"approved":    approved,
			"status":      status,
			"responded_at": respondedAt,
		},
	}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments // Or a custom not found error
	}
	return nil
}

// MongoGetPendingRequests retrieves all HITL requests with "pending" status.
// Consider if filtering by clientID or other fields is needed.
func MongoGetPendingRequests(db *mongo.Database) ([]*types.HITLRequest, error) {
	collection := db.Collection(hitlRequestsCollectionName)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"status": types.RequestStatusPending}
	cursor, err := collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var requests []*types.HITLRequest
	if err = cursor.All(ctx, &requests); err != nil {
		return nil, err
	}
	return requests, nil
}

// MongoCancelRequest updates the status of a HITL request to "canceled".
func MongoCancelRequest(db *mongo.Database, requestID string) error {
	collection := db.Collection(hitlRequestsCollectionName)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": requestID}
	update := bson.M{"$set": bson.M{"status": types.RequestStatusCanceled}}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments // Or a custom not found error
	}
	return nil
}

// MongoRegisterSession stores a new session in MongoDB.
// If a session with the same ID already exists, it can either error or update it (upsert).
// Current implementation will error if _id (SessionID) conflicts.
func MongoRegisterSession(db *mongo.Database, session *types.Session) error {
	collection := db.Collection(sessionsCollectionName)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := collection.InsertOne(ctx, session)
	// Consider an upsert if re-registering a session should update it
	// opts := options.Update().SetUpsert(true)
	// _, err := collection.UpdateOne(ctx, bson.M{"_id": session.ID}, bson.M{"$set": session}, opts)
	return err
}

// MongoGetSession retrieves a session by its ID from MongoDB.
func MongoGetSession(db *mongo.Database, sessionID string) (*types.Session, error) {
	collection := db.Collection(sessionsCollectionName)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var session types.Session
	err := collection.FindOne(ctx, bson.M{"_id": sessionID}).Decode(&session)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, mongo.ErrNoDocuments // Return mongo.ErrNoDocuments
		}
		return nil, err
	}
	return &session, nil
}

// MongoDeactivateSession updates the status of a session to "active: false".
func MongoDeactivateSession(db *mongo.Database, sessionID string) error {
	collection := db.Collection(sessionsCollectionName)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": sessionID}
	update := bson.M{"$set": bson.M{"active": false}}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments // Or a custom not found error
	}
	return nil
}

// MongoGetTelegramIDForClient retrieves the TelegramID for an active session associated with a clientID.
// This assumes a clientID can have multiple sessions, but we're interested in an active one.
// If multiple active sessions exist for a clientID, this returns the first one found.
// The logic might need refinement based on how clientID/sessionID uniqueness is handled.
func MongoGetTelegramIDForClient(db *mongo.Database, clientID string) (int64, error) {
	collection := db.Collection(sessionsCollectionName)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"client_id": clientID, "active": true}
	var session types.Session
	err := collection.FindOne(ctx, filter).Decode(&session)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return 0, errors.New("active session not found for client") // Specific error
		}
		return 0, err
	}
	return session.TelegramID, nil
}

// MongoGetActiveSessions retrieves all sessions with "active: true".
func MongoGetActiveSessions(db *mongo.Database) ([]*types.Session, error) {
	collection := db.Collection(sessionsCollectionName)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"active": true}
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var sessions []*types.Session
	if err = cursor.All(ctx, &sessions); err != nil {
		return nil, err
	}
	return sessions, nil
}

// Helper for MongoGetTelegramIDForClient - use standard errors package
// type Error string
// func (e Error) Error() string { return string(e) }
// const ErrNotFound = Error("active session not found for client")
