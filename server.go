package webwire

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
)

const protocolVersion = "1.4"

// server represents a headless WebWire server instance,
// where headless means there's no HTTP server that's hosting it
type server struct {
	impl              ServerImplementation
	httpServer        *http.Server
	listener          net.Listener
	sessionManager    SessionManager
	sessionKeyGen     SessionKeyGenerator
	sessionInfoParser SessionInfoParser

	// State
	addr            net.Addr
	options         ServerOptions
	shutdown        bool
	shutdownRdy     chan bool
	currentOps      uint32
	opsLock         *sync.Mutex
	connectionsLock *sync.Mutex
	connections     []*connection
	sessionsEnabled bool
	sessionRegistry *sessionRegistry

	// Internals
	connUpgrader ConnUpgrader
	warnLog      *log.Logger
	errorLog     *log.Logger
}

func (srv *server) shutdownHTTPServer() error {
	if srv.httpServer == nil {
		return nil
	}
	if err := srv.httpServer.Shutdown(context.Background()); err != nil {
		return fmt.Errorf("Couldn't properly shutdown HTTP server: %s", err)
	}
	return nil
}

// Run implements the Server interface
func (srv *server) Run() error {
	// Launch HTTP server
	if err := srv.httpServer.Serve(
		tcpKeepAliveListener{srv.listener.(*net.TCPListener)},
	); err != http.ErrServerClosed {
		return fmt.Errorf("HTTP Server failure: %s", err)
	}

	return nil
}

// Addr implements the Server interface
func (srv *server) Addr() net.Addr {
	return srv.addr
}

// Shutdown implements the Server interface
func (srv *server) Shutdown() error {
	srv.opsLock.Lock()
	srv.shutdown = true
	// Don't block if there's no currently processed operations
	if srv.currentOps < 1 {
		srv.opsLock.Unlock()
		return srv.shutdownHTTPServer()
	}
	srv.opsLock.Unlock()
	<-srv.shutdownRdy

	return srv.shutdownHTTPServer()
}

// ActiveSessionsNum implements the Server interface
func (srv *server) ActiveSessionsNum() int {
	return srv.sessionRegistry.activeSessionsNum()
}

// SessionConnectionsNum implements the Server interface
func (srv *server) SessionConnectionsNum(sessionKey string) int {
	return srv.sessionRegistry.sessionConnectionsNum(sessionKey)
}

// SessionConnections implements the Server interface
func (srv *server) SessionConnections(sessionKey string) []Connection {
	connections := srv.sessionRegistry.sessionConnections(sessionKey)
	if connections == nil {
		return nil
	}
	list := make([]Connection, len(connections))
	i := 0
	for connection := range connections {
		list[i] = connection
		i++
	}
	return list
}

// CloseSession implements the Server interface
func (srv *server) CloseSession(sessionKey string) (
	affectedConnections []Connection,
	errors []error,
	generalError error,
) {
	connections := srv.sessionRegistry.sessionConnections(sessionKey)

	errors = make([]error, len(connections))
	if connections == nil {
		return nil, nil, nil
	}
	affectedConnections = make([]Connection, len(connections))
	i := 0
	errNum := 0
	for connection := range connections {
		affectedConnections[i] = connection
		err := connection.CloseSession()
		if err != nil {
			errors[i] = err
			errNum++
		} else {
			errors[i] = nil
		}
		i++
	}

	if errNum > 0 {
		generalError = fmt.Errorf(
			"%d errors during the closure of a session",
			errNum,
		)
	}

	return affectedConnections, errors, generalError
}
