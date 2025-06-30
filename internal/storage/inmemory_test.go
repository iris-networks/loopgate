package storage

import (
	"loopgate/internal/types"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryStorageAdapter_SessionManagement(t *testing.T) {
	adapter := NewInMemoryStorageAdapter()

	sessionID := "test-session-1"
	clientID := "test-client-1"
	telegramID := int64(12345)

	// Test RegisterSession
	err := adapter.RegisterSession(sessionID, clientID, telegramID)
	require.NoError(t, err)

	// Test GetSession
	session, err := adapter.GetSession(sessionID)
	require.NoError(t, err)
	require.NotNil(t, session)
	assert.Equal(t, sessionID, session.ID)
	assert.Equal(t, clientID, session.ClientID)
	assert.Equal(t, telegramID, session.TelegramID)
	assert.True(t, session.Active)

	// Test GetTelegramID
	retrievedTelegramID, err := adapter.GetTelegramID(clientID)
	require.NoError(t, err)
	assert.Equal(t, telegramID, retrievedTelegramID)

	// Test DeactivateSession
	err = adapter.DeactivateSession(sessionID)
	require.NoError(t, err)
	session, err = adapter.GetSession(sessionID)
	require.NoError(t, err)
	require.NotNil(t, session)
	assert.False(t, session.Active)

	// Test GetTelegramID for deactivated session's client (should fail or return error)
	_, err = adapter.GetTelegramID(clientID)
	assert.Error(t, err, "Expected error when getting TelegramID for client with deactivated session")


	// Test GetActiveSessions
	activeSessions, err := adapter.GetActiveSessions()
	require.NoError(t, err)
	assert.Empty(t, activeSessions, "Expected no active sessions after deactivation")

	// Register another active session to test GetActiveSessions
	err = adapter.RegisterSession("active-session-2", "client-2", 67890)
	require.NoError(t, err)
	activeSessions, err = adapter.GetActiveSessions()
	require.NoError(t, err)
	assert.Len(t, activeSessions, 1, "Expected one active session")
	assert.Equal(t, "active-session-2", activeSessions[0].ID)
}

func TestInMemoryStorageAdapter_RequestManagement(t *testing.T) {
	adapter := NewInMemoryStorageAdapter()
	requestID := "test-request-1"
	sessionID := "test-session-for-request"

	request := &types.HITLRequest{
		ID:        requestID,
		SessionID: sessionID,
		ClientID:  "client-req-1",
		Message:   "Test request message",
		Status:    types.RequestStatusPending,
		CreatedAt: time.Now(),
	}

	// Test StoreRequest
	err := adapter.StoreRequest(request)
	require.NoError(t, err)

	// Test GetRequest
	retrievedRequest, err := adapter.GetRequest(requestID)
	require.NoError(t, err)
	require.NotNil(t, retrievedRequest)
	assert.Equal(t, requestID, retrievedRequest.ID)
	assert.Equal(t, types.RequestStatusPending, retrievedRequest.Status)

	// Test GetPendingRequests
	pendingRequests, err := adapter.GetPendingRequests()
	require.NoError(t, err)
	require.Len(t, pendingRequests, 1)
	assert.Equal(t, requestID, pendingRequests[0].ID)

	// Test UpdateRequestResponse
	responseMessage := "This is the response"
	err = adapter.UpdateRequestResponse(requestID, responseMessage, true)
	require.NoError(t, err)

	updatedRequest, err := adapter.GetRequest(requestID)
	require.NoError(t, err)
	assert.Equal(t, types.RequestStatusCompleted, updatedRequest.Status)
	assert.Equal(t, responseMessage, updatedRequest.Response)
	assert.True(t, updatedRequest.Approved)
	assert.NotNil(t, updatedRequest.RespondedAt)

	// Test GetPendingRequests after update (should be empty)
	pendingRequests, err = adapter.GetPendingRequests()
	require.NoError(t, err)
	assert.Empty(t, pendingRequests)

	// Test CancelRequest
	requestToCancelID := "request-to-cancel"
	requestToCancel := &types.HITLRequest{
		ID:        requestToCancelID,
		SessionID: sessionID,
		Status:    types.RequestStatusPending,
	}
	err = adapter.StoreRequest(requestToCancel)
	require.NoError(t, err)

	err = adapter.CancelRequest(requestToCancelID)
	require.NoError(t, err)
	cancelledRequest, err := adapter.GetRequest(requestToCancelID)
	require.NoError(t, err)
	assert.Equal(t, types.RequestStatusCanceled, cancelledRequest.Status)
}

func TestInMemoryStorageAdapter_ErrorConditions(t *testing.T) {
	adapter := NewInMemoryStorageAdapter()
	// Using a different error variable name to be absolutely sure about scoping.
	var errCond error

	// Test GetSession for non-existent session
	_, errCond = adapter.GetSession("non-existent-session")
	assert.Error(t, errCond)

	// Test GetRequest for non-existent request
	_, errCond = adapter.GetRequest("non-existent-request")
	assert.Error(t, errCond)

	// Test DeactivateSession for non-existent session
	errCond = adapter.DeactivateSession("non-existent-session")
	assert.Error(t, errCond)

	// Test UpdateRequestResponse for non-existent request
	errCond = adapter.UpdateRequestResponse("non-existent-request", "response", true)
	assert.Error(t, errCond)

	// Test CancelRequest for non-existent request
	errCond = adapter.CancelRequest("non-existent-request")
	assert.Error(t, errCond)

	// Test RegisterSession with existing ID
	errCond = adapter.RegisterSession("duplicate-session-id", "client1-dup", 11122)
	require.NoError(t, errCond, "First registration of 'duplicate-session-id' should succeed")
	errCond = adapter.RegisterSession("duplicate-session-id", "client2-dup", 33344) // Attempt to register same ID again
	assert.Error(t, errCond, "Should error when registering session with duplicate ID")

	// Test StoreRequest with existing ID
	reqOriginal := &types.HITLRequest{ID: "duplicate-req-id", SessionID: "s1", Status: types.RequestStatusPending, CreatedAt: time.Now()}
	errCond = adapter.StoreRequest(reqOriginal)
	require.NoError(t, errCond, "First store of 'duplicate-req-id' should succeed")
	reqDuplicate := &types.HITLRequest{ID: "duplicate-req-id", SessionID: "s2", Status: types.RequestStatusPending, CreatedAt: time.Now()}
	errCond = adapter.StoreRequest(reqDuplicate) // Attempt to store same ID again
	assert.Error(t, errCond, "Should error when storing request with duplicate ID")

	// Test GetTelegramID for non-existent client
	_, errCond = adapter.GetTelegramID("non-existent-client")
	assert.Error(t, errCond)
}
