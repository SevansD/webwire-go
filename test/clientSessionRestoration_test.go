package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	webwire "github.com/qbeon/webwire-go"
	webwireClient "github.com/qbeon/webwire-go/client"
)

// TestClientSessionRestoration verifies manual session restoration from a session key
func TestClientSessionRestoration(t *testing.T) {
	sessionStorage := make(map[string]*webwire.Session)

	currentStep := 1
	var createdSession *webwire.Session

	// Initialize webwire server
	_, addr := setupServer(
		t,
		webwire.ServerOptions{
			SessionsEnabled: true,
			SessionManager: &CallbackPoweredSessionManager{
				// Saves the session
				SessionCreated: func(client *webwire.Client) error {
					sess := client.Session()
					sessionStorage[sess.Key] = sess
					return nil
				},
				// Finds session by key
				SessionLookup: func(key string) (*webwire.Session, error) {
					// Expect the key of the created session to be looked up
					if key != createdSession.Key {
						err := fmt.Errorf(
							"Expected and looked up session keys differ: %s | %s",
							createdSession.Key,
							key,
						)
						t.Fatalf("Session lookup mismatch: %s", err)
						return nil, err
					}

					if session, exists := sessionStorage[key]; exists {
						return session, nil
					}

					// Expect the session to be found
					t.Fatalf(
						"Expected session (%s) not found in: %v",
						createdSession.Key,
						sessionStorage,
					)
					return nil, nil
				},
			},
			Hooks: webwire.Hooks{
				OnRequest: func(ctx context.Context) (webwire.Payload, error) {
					// Extract request message and requesting client from the context
					msg := ctx.Value(webwire.Msg).(webwire.Message)

					if currentStep == 2 {
						// Expect the session to have been automatically restored
						compareSessions(t, createdSession, msg.Client.Session())
						return webwire.Payload{}, nil
					}

					// Try to create a new session
					if err := msg.Client.CreateSession(nil); err != nil {
						return webwire.Payload{}, err
					}

					// Return the key of the newly created session
					return webwire.Payload{}, nil
				},
			},
		},
	)

	// Initialize client
	initialClient := webwireClient.NewClient(
		addr,
		webwireClient.Options{
			DefaultRequestTimeout: 2 * time.Second,
		},
	)

	if err := initialClient.Connect(); err != nil {
		t.Fatalf("Couldn't connect initial client: %s", err)
	}

	/*****************************************************************\
		Step 1 - Create session and disconnect
	\*****************************************************************/

	// Create a new session
	if _, err := initialClient.Request(
		"login",
		webwire.Payload{Data: []byte("auth")},
	); err != nil {
		t.Fatalf("Auth request failed: %s", err)
	}

	tmp := initialClient.Session()
	createdSession = &tmp

	// Disconnect client without closing the session
	initialClient.Close()

	/*****************************************************************\
		Step 2 - Create new client, restore session from key
	\*****************************************************************/
	currentStep = 2

	// Initialize client
	secondClient := webwireClient.NewClient(
		addr,
		webwireClient.Options{
			DefaultRequestTimeout: 2 * time.Second,
		},
	)

	if err := secondClient.Connect(); err != nil {
		t.Fatalf("Couldn't connect second client: %s", err)
	}

	// Ensure there's no active session on the second client
	sessionBefore := secondClient.Session()
	if sessionBefore.Key != "" {
		t.Fatalf("Expected the second client to have no session, got: %v", sessionBefore)
	}

	// Try to manually restore the session using the initial clients session key
	if err := secondClient.RestoreSession([]byte(createdSession.Key)); err != nil {
		t.Fatalf("Manual session restoration failed: %s", err)
	}

	// Verify session
	sessionAfter := secondClient.Session()
	if sessionAfter.Key == "" {
		t.Fatalf("Expected the second client to have an active session, got: %v", sessionAfter)
	}
	compareSessions(t, createdSession, &sessionAfter)
}
