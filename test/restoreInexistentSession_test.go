package test

import (
	"reflect"
	"testing"
	"time"

	wwr "github.com/qbeon/webwire-go"
	wwrClient "github.com/qbeon/webwire-go/client"
)

// TestRestoreInexistentSession tests the restoration of an inexistent session
func TestRestoreInexistentSession(t *testing.T) {
	// Initialize server
	_, addr := setupServer(
		t,
		wwr.ServerOptions{
			SessionsEnabled: true,
		},
	)

	// Initialize client

	// Ensure that the last superfluous client is rejected
	client := wwrClient.NewClient(
		addr,
		wwrClient.Options{
			DefaultRequestTimeout: 2 * time.Second,
		},
	)

	if err := client.Connect(); err != nil {
		t.Fatalf("Couldn't connect client: %s", err)
	}

	// Try to restore the session and expect it to fail due to the session being inexistent
	sessRestErr := client.RestoreSession([]byte("lalala"))
	if _, isSessNotFoundErr := sessRestErr.(wwr.SessNotFoundErr); !isSessNotFoundErr {
		t.Fatalf(
			"Expected a SessNotFound error during manual session restoration, got: %s | %s",
			reflect.TypeOf(sessRestErr),
			sessRestErr,
		)
	}
}
