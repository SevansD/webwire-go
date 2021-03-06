package test

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tmdwg "github.com/qbeon/tmdwg-go"
	wwr "github.com/qbeon/webwire-go"
	wwrclt "github.com/qbeon/webwire-go/client"
)

// TestClientDisconnectedHook verifies the server is calling the
// onClientDisconnected hook properly
func TestClientDisconnectedHook(t *testing.T) {
	disconnectedHookCalled := tmdwg.NewTimedWaitGroup(1, 1*time.Second)
	var clientConn wwr.Connection
	connectedClientLock := sync.Mutex{}

	// Initialize webwire server given only the request
	server := setupServer(
		t,
		&serverImpl{
			onClientConnected: func(conn wwr.Connection) {
				connectedClientLock.Lock()
				clientConn = conn
				connectedClientLock.Unlock()
			},
			onClientDisconnected: func(conn wwr.Connection) {
				connectedClientLock.Lock()
				assert.Equal(t, clientConn, conn)
				connectedClientLock.Unlock()
				disconnectedHookCalled.Progress(1)
			},
		},
		wwr.ServerOptions{},
	)

	// Initialize client
	client := newCallbackPoweredClient(
		server.Addr().String(),
		wwrclt.Options{
			DefaultRequestTimeout: 2 * time.Second,
		},
		callbackPoweredClientHooks{},
	)

	// Connect to the server
	require.NoError(t, client.connection.Connect())

	// Disconnect the client
	client.connection.Close()

	// Await the onClientDisconnected hook to be called on the server
	require.NoError(t,
		disconnectedHookCalled.Wait(),
		"server.OnClientDisconnected hook not called",
	)
}
