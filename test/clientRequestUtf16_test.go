package test

import (
	"context"
	"testing"
	"time"

	webwire "github.com/qbeon/webwire-go"
	webwireClient "github.com/qbeon/webwire-go/client"
)

// TestClientRequestUtf16 tests requests with UTF16 encoded payloads
func TestClientRequestUtf16(t *testing.T) {
	testPayload := webwire.Payload{
		Encoding: webwire.EncodingUtf16,
		Data:     []byte{00, 115, 00, 97, 00, 109, 00, 112, 00, 108, 00, 101},
	}
	verifyPayload := func(payload webwire.Payload) {
		if payload.Encoding != webwire.EncodingUtf16 {
			t.Errorf("Unexpected payload encoding: %s", payload.Encoding.String())
		}
		if len(testPayload.Data) != len(payload.Data) {
			t.Errorf("Corrupt payload: %s", payload.Data)
		}
		for i := 0; i < len(testPayload.Data); i++ {
			if testPayload.Data[i] != payload.Data[i] {
				t.Errorf("Corrupt payload, mismatching byte at position %d: %s", i, payload.Data)
				return
			}
		}
	}

	// Initialize webwire server given only the request
	_, addr := setupServer(
		t,
		webwire.ServerOptions{
			Hooks: webwire.Hooks{
				OnRequest: func(ctx context.Context) (webwire.Payload, error) {
					// Extract request message from the context
					msg := ctx.Value(webwire.Msg).(webwire.Message)

					verifyPayload(msg.Payload)

					return webwire.Payload{
						Encoding: webwire.EncodingUtf16,
						Data:     []byte{00, 115, 00, 97, 00, 109, 00, 112, 00, 108, 00, 101},
					}, nil
				},
			},
		},
	)

	// Initialize client
	client := webwireClient.NewClient(
		addr,
		webwireClient.Options{
			DefaultRequestTimeout: 2 * time.Second,
		},
	)

	if err := client.Connect(); err != nil {
		t.Fatalf("Couldn't connect: %s", err)
	}

	// Send request and await reply
	reply, err := client.Request("", webwire.Payload{
		Encoding: webwire.EncodingUtf16,
		Data:     []byte{00, 115, 00, 97, 00, 109, 00, 112, 00, 108, 00, 101},
	})
	if err != nil {
		t.Fatalf("Request failed: %s", err)
	}

	// Verify reply
	verifyPayload(reply)
}
