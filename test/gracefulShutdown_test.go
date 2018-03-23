package test

import (
	"context"
	"testing"
	"time"

	wwr "github.com/qbeon/webwire-go"
	wwrclt "github.com/qbeon/webwire-go/client"
)

// TestGracefulShutdown tests the ability of the server to delay shutdown
// until all requests and signals are processed and reject incoming connections and requests
// while ignoring incoming signals
//
// SIGNAL:       --->||||||||||----------------- (must finish)
// REQUEST:      ---->||||||||||---------------- (must finish)
// SRV SHUTDWN:  -------->||||||---------------- (must await req and sig)
// LATE CONN:    ---------->|------------------- (must be rejected)
// LATE REQ:     ----------->|------------------ (must be rejected)
func TestGracefulShutdown(t *testing.T) {
	expectedReqReply := []byte("ifinished")
	timeDelta := time.Duration(1)
	processesFinished := NewPending(2, 1*time.Second, true)
	serverShutDown := NewPending(1, timeDelta*500*time.Millisecond, true)

	// Initialize webwire server
	server, addr := setupServer(
		t,
		wwr.ServerOptions{
			Hooks: wwr.Hooks{
				OnSignal: func(ctx context.Context) {
					time.Sleep(timeDelta * 100 * time.Millisecond)
					processesFinished.Done()
				},
				OnRequest: func(ctx context.Context) (wwr.Payload, error) {
					time.Sleep(timeDelta * 100 * time.Millisecond)
					return wwr.Payload{Data: expectedReqReply}, nil
				},
			},
		},
	)

	// Initialize different clients for the signal, the request and the late request and conn
	// to avoid serializing them because every client is handled in a separate goroutine
	cltOpts := wwrclt.Options{
		Hooks: wwrclt.Hooks{},
		DefaultRequestTimeout: 5 * time.Second,
	}
	clientSig := wwrclt.NewClient(addr, cltOpts)
	clientReq := wwrclt.NewClient(addr, cltOpts)
	clientLateReq := wwrclt.NewClient(addr, cltOpts)

	// Disable autoconnect for the late client to enable immediate errors
	clientLateConn := wwrclt.NewClient(addr, wwrclt.Options{
		Autoconnect: wwrclt.OptDisabled,
	})

	if err := clientSig.Connect(); err != nil {
		t.Fatalf("Couldn't connect signal client: %s", err)
	}
	if err := clientReq.Connect(); err != nil {
		t.Fatalf("Couldn't connect request client: %s", err)
	}
	if err := clientLateReq.Connect(); err != nil {
		t.Fatalf("Couldn't connect late-request client: %s", err)
	}

	// Send signal and request in another parallel goroutine
	// to avoid blocking the main test goroutine when awaiting the request reply
	go func() {
		// (SIGNAL)
		if err := clientSig.Signal("", wwr.Payload{Data: []byte("test")}); err != nil {
			t.Errorf("Signal failed: %s", err)
		}

		// (REQUEST)
		if rep, err := clientReq.Request("", wwr.Payload{Data: []byte("test")}); err != nil {
			t.Errorf("Request failed: %s", err)
		} else if string(rep.Data) != string(expectedReqReply) {
			t.Errorf(
				"Expected and actual replies differ: %s | %s",
				string(expectedReqReply),
				string(rep.Data),
			)
		}
	}()

	// Request server shutdown in another parallel goroutine
	// to avoid blocking the main test goroutine when waiting for the server to shut down
	go func() {
		// Wait for the signal and request to arrive and get handled, then request the shutdown
		time.Sleep(timeDelta * 10 * time.Millisecond)
		// (SRV SHUTDWN)
		server.Shutdown()
		serverShutDown.Done()
	}()

	// Fire late requests and late connection in another parallel goroutine
	// to avoid blocking the main test goroutine when performing them
	go func() {
		// Wait for the server to start shutting down
		time.Sleep(timeDelta * 20 * time.Millisecond)

		// Verify connection establishment during shutdown (LATE CONN)
		if err := clientLateConn.Connect(); err == nil {
			t.Errorf("Expected late connection to be rejected, though it still was accepted")
		}

		// Verify request rejection during shutdown (LATE REQ)
		_, lateReqErr := clientLateReq.Request("", wwr.Payload{Data: []byte("test")})
		switch err := lateReqErr.(type) {
		case wwr.ReqSrvShutdownErr:
			break
		case wwr.ReqErr:
			t.Errorf("Expected special server shutdown error, got regular request error: %s", err)
		default:
			t.Errorf("Expected request during shutdown to be rejected with special error type")
		}

		processesFinished.Done()
	}()

	// Await server shutdown, timeout if necessary
	if err := serverShutDown.Wait(); err != nil {
		t.Fatalf("Expected server to shut down within n seconds")
	}

	// Expect both the signal and the request to have completed properly
	if err := processesFinished.Wait(); err != nil {
		t.Fatalf("Expected signal and request to have finished processing")
	}
}
