package storage

import (
	"fmt"
	"loopgate/internal/types"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testSQLiteDSN returns a DSN for a temporary SQLite database file.
func testSQLiteDSN(t *testing.T) string {
	t.Helper()
	// Using a unique file per test run to avoid conflicts,
	// especially if tests run in parallel or if previous cleanup failed.
	return fmt.Sprintf("file:test_loopgate_%d.db?cache=shared&mode=memory", time.Now().UnixNano())
	// For a file-based DB for inspection:
	// return "test_loopgate.db"
}

func setupSQLiteAdapter(t *testing.T) (*SQLiteStorageAdapter, func()) {
	t.Helper()
	dsn := testSQLiteDSN(t)
	adapter, err := NewSQLiteStorageAdapter(dsn)
	require.NoError(t, err, "Failed to create SQLite adapter for testing")

	cleanup := func() {
		err := adapter.Close()
		assert.NoError(t, err, "Failed to close SQLite adapter")
		// If using a file-based DSN for testing, uncomment to clean up:
		// if dsn != "file::memory:?cache=shared" {
		// 	os.Remove(dsn)
		// }
	}

	return adapter, cleanup
}

func TestSQLiteStorageAdapter_SessionManagement(t *testing.T) {
	adapter, cleanup := setupSQLiteAdapter(t)
	defer cleanup()

	sessionID := "test-session-sqlite-1"
	clientID := "test-client-sqlite-1"
	telegramID := int64(98765)

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
	assert.WithinDuration(t, time.Now(), session.CreatedAt, 5*time.Second, "CreatedAt should be recent")


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

	// Test GetTelegramID for client with deactivated session
	_, err = adapter.GetTelegramID(clientID)
	assert.Error(t, err, "Expected error when getting TelegramID for client with no active session")


	// Test GetActiveSessions
	activeSessions, err := adapter.GetActiveSessions()
	require.NoError(t, err)
	assert.Empty(t, activeSessions, "Expected no active sessions after deactivation")

	// Register another active session
	err = adapter.RegisterSession("active-session-sqlite-2", "client-sqlite-2", 67890)
	require.NoError(t, err)
	activeSessions, err = adapter.GetActiveSessions()
	require.NoError(t, err)
	assert.Len(t, activeSessions, 1, "Expected one active session")
	assert.Equal(t, "active-session-sqlite-2", activeSessions[0].ID)
}

func TestSQLiteStorageAdapter_RequestManagement(t *testing.T) {
	adapter, cleanup := setupSQLiteAdapter(t)
	defer cleanup()

	requestID := "test-request-sqlite-1"
	sessionID := "test-session-for-request-sqlite"

	// Need to register the session first due to potential foreign key constraints if implemented
	err := adapter.RegisterSession(sessionID, "client-for-req-sqlite", 111222)
	require.NoError(t, err)


	request := &types.HITLRequest{
		ID:          requestID,
		SessionID:   sessionID,
		ClientID:    "client-req-sqlite-1",
		Message:     "Test SQLite request message",
		Status:      types.RequestStatusPending,
		CreatedAt:   time.Now().UTC().Truncate(time.Second), // Truncate for DB precision
		RequestType: types.RequestTypeConfirmation,
	}

	// Test StoreRequest
	err = adapter.StoreRequest(request)
	require.NoError(t, err)

	// Test GetRequest
	retrievedRequest, err := adapter.GetRequest(requestID)
	require.NoError(t, err)
	require.NotNil(t, retrievedRequest)
	assert.Equal(t, requestID, retrievedRequest.ID)
	assert.Equal(t, types.RequestStatusPending, retrievedRequest.Status)
	assert.Equal(t, request.Message, retrievedRequest.Message)
	assert.WithinDuration(t, request.CreatedAt, retrievedRequest.CreatedAt, time.Second, "CreatedAt mismatch")


	// Test GetPendingRequests
	pendingRequests, err := adapter.GetPendingRequests()
	require.NoError(t, err)
	require.Len(t, pendingRequests, 1)
	assert.Equal(t, requestID, pendingRequests[0].ID)

	// Test UpdateRequestResponse
	responseMessage := "This is the SQLite response"
	err = adapter.UpdateRequestResponse(requestID, responseMessage, true)
	require.NoError(t, err)

	updatedRequest, err := adapter.GetRequest(requestID)
	require.NoError(t, err)
	assert.Equal(t, types.RequestStatusCompleted, updatedRequest.Status)
	assert.Equal(t, responseMessage, updatedRequest.Response)
	assert.True(t, updatedRequest.Approved)
	require.NotNil(t, updatedRequest.RespondedAt)
	assert.WithinDuration(t, time.Now(), *updatedRequest.RespondedAt, 5*time.Second, "RespondedAt should be recent")


	// Test GetPendingRequests after update
	pendingRequests, err = adapter.GetPendingRequests()
	require.NoError(t, err)
	assert.Empty(t, pendingRequests)

	// Test CancelRequest
	requestToCancelID := "request-to-cancel-sqlite"
	requestToCancel := &types.HITLRequest{
		ID:        requestToCancelID,
		SessionID: sessionID,
		Status:    types.RequestStatusPending,
		CreatedAt: time.Now().UTC().Truncate(time.Second),
	}
	err = adapter.StoreRequest(requestToCancel)
	require.NoError(t, err)

	err = adapter.CancelRequest(requestToCancelID)
	require.NoError(t, err)
	cancelledRequest, err := adapter.GetRequest(requestToCancelID)
	require.NoError(t, err)
	assert.Equal(t, types.RequestStatusCanceled, cancelledRequest.Status)
}

func TestSQLiteStorageAdapter_ErrorConditions(t *testing.T) {
	adapter, cleanup := setupSQLiteAdapter(t)
	defer cleanup()

	// Test GetSession for non-existent session
	_, err := adapter.GetSession("non-existent-session-sqlite")
	assert.Error(t, err) // GORM returns gorm.ErrRecordNotFound, which our adapter maps to "session not found"

	// Test GetRequest for non-existent request
	_, err = adapter.GetRequest("non-existent-request-sqlite")
	assert.Error(t, err) // GORM returns gorm.ErrRecordNotFound, which our adapter maps to "request not found"

	// Test DeactivateSession for non-existent session - GORM might not error if record not found for update
	// This depends on GORM's behavior, often it's RowsAffected == 0, not an error.
	// Let's check if it errors as per our current adapter logic for other dbs (it should not for GORM update if not found)
	// For consistency, our adapter should probably return an error.
	// Current GORM: s.db.Model(&types.Session{}).Where("id = ?", sessionID).Update("active", false).Error returns nil if not found.
	// We might need to check RowsAffected or do a find first if strict error is needed.
	// For now, we test the current behavior.
	err = adapter.DeactivateSession("non-existent-session-sqlite")
	assert.NoError(t, err, "Deactivate on non-existent session should ideally not error with GORM update, or we should add a check")


	// Test UpdateRequestResponse for non-existent request
	err = adapter.UpdateRequestResponse("non-existent-request-sqlite", "response", true)
	assert.Error(t, err) // This should error because GetRequest inside it will fail

	// Test CancelRequest for non-existent request - similar to DeactivateSession with GORM update
	err = adapter.CancelRequest("non-existent-request-sqlite")
	assert.NoError(t, err, "Cancel on non-existent request should ideally not error with GORM update, or we should add a check")

	// Test RegisterSession with existing ID (Primary Key violation)
	err = adapter.RegisterSession("existing-id-sqlite", "client1-s", 111)
	require.NoError(t, err)
	err = adapter.RegisterSession("existing-id-sqlite", "client2-s", 222)
	assert.Error(t, err, "Should not allow registering session with duplicate ID (PK violation)")

	// Test StoreRequest with existing ID (Primary Key violation)
	err = adapter.StoreRequest(&types.HITLRequest{ID: "existing-req-sqlite", SessionID: "s1", CreatedAt: time.Now()})
	require.NoError(t, err)
	err = adapter.StoreRequest(&types.HITLRequest{ID: "existing-req-sqlite", SessionID: "s2", CreatedAt: time.Now()})
	assert.Error(t, err, "Should not allow storing request with duplicate ID (PK violation)")

	// Test GetTelegramID for non-existent client
	_, err = adapter.GetTelegramID("non-existent-client-sqlite")
	assert.Error(t, err)
}

// TestSQLiteStorageAdapter_Persistence ensures data persists if not using in-memory.
// This test is more relevant if using a file-based SQLite DB for testing.
// For "file::memory:?cache=shared", data persists across connections in the same process.
func TestSQLiteStorageAdapter_Persistence(t *testing.T) {
	dsn := "test_loopgate_persistence.db" // Use a real file for this test
	defer os.Remove(dsn)                 // Clean up the file afterwards

	adapter1, err := NewSQLiteStorageAdapter(dsn)
	require.NoError(t, err)

	sessionID := "persistent-session-1"
	clientID := "persistent-client-1"
	telegramID := int64(100100)

	err = adapter1.RegisterSession(sessionID, clientID, telegramID)
	require.NoError(t, err)
	err = adapter1.Close()
	require.NoError(t, err)

	// New adapter instance, should connect to the same DB file
	adapter2, err := NewSQLiteStorageAdapter(dsn)
	require.NoError(t, err)
	defer adapter2.Close()

	session, err := adapter2.GetSession(sessionID)
	require.NoError(t, err)
	require.NotNil(t, session)
	assert.Equal(t, clientID, session.ClientID)
}
