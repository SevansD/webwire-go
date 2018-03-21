package test

import (
	"reflect"
	"testing"
	"time"

	wwr "github.com/qbeon/webwire-go"
	wwrclt "github.com/qbeon/webwire-go/client"
)

// TestClientRestSessDisconnNoAutoconn tests disconnected error when trying to manually
// restore the session while the server is unreachable and autoconn is disabled
func TestClientRestSessDisconnNoAutoconn(t *testing.T) {
	// Initialize client
	client := wwrclt.NewClient(
		"127.0.0.1:65000",
		wwrclt.Options{
			Autoconnect:           wwrclt.OptDisabled,
			ReconnectionInterval:  5 * time.Millisecond,
			DefaultRequestTimeout: 50 * time.Millisecond,
		},
	)

	// Send request and await reply
	err := client.RestoreSession([]byte("inexistentkey"))
	if _, isDisconnErr := err.(wwr.DisconnectedErr); !isDisconnErr {
		t.Fatalf(
			"Expected disconnected error error, got: %s | %s",
			reflect.TypeOf(err),
			err,
		)
	}
}
