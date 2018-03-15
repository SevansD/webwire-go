package test

import (
	"testing"
	"time"

	webwire "github.com/qbeon/webwire-go"
	webwireClient "github.com/qbeon/webwire-go/client"
)

// TestClientConcurrentConnect verifies concurrent calling of client.Connect
// is properly synchronized and doesn't cause any data race
func TestClientConcurrentConnect(t *testing.T) {
	var concurrentAccessors uint32 = 16
	finished := NewPending(concurrentAccessors, 2*time.Second, true)

	// Initialize webwire server
	_, addr := setupServer(t, webwire.ServerOptions{})

	// Initialize client
	client := webwireClient.NewClient(
		addr,
		webwireClient.Options{
			DefaultRequestTimeout: 2 * time.Second,
		},
	)
	defer client.Close()

	connect := func() {
		defer finished.Done()
		if err := client.Connect(); err != nil {
			t.Errorf("Connect failed: %s", err)
		}
	}

	for i := uint32(0); i < concurrentAccessors; i++ {
		go connect()
	}

	if err := finished.Wait(); err != nil {
		t.Fatal("Expectation timed out")
	}
}
