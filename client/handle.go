package client

import (
	"encoding/json"

	webwire "github.com/qbeon/webwire-go"
)

func (clt *Client) handleSessionCreated(sessionKey []byte) {
	// Set new session
	var session webwire.Session

	if err := json.Unmarshal(sessionKey, &session); err != nil {
		clt.errorLog.Printf("Failed unmarshalling session object: %s", err)
		return
	}

	clt.sessionLock.Lock()
	clt.session = &session
	clt.sessionLock.Unlock()
	clt.hooks.OnSessionCreated(&session)
}

func (clt *Client) handleSessionClosed() {
	// Destroy local session
	clt.sessionLock.Lock()
	clt.session = nil
	clt.sessionLock.Unlock()

	clt.hooks.OnSessionClosed()
}

func (clt *Client) handleFailure(reqID [8]byte, payload []byte) {
	// Decode error
	var replyErr webwire.ReqErr
	if err := json.Unmarshal(payload, &replyErr); err != nil {
		clt.errorLog.Printf("Failed unmarshalling error reply: %s", err)
	}

	// Fail request
	clt.requestManager.Fail(reqID, replyErr)
}

func (clt *Client) handleInternalError(reqIdent [8]byte) {
	// Fail request
	clt.requestManager.Fail(reqIdent, webwire.ReqInternalErr{})
}

func (clt *Client) handleReplyShutdown(reqIdent [8]byte) {
	clt.requestManager.Fail(reqIdent, webwire.ReqSrvShutdownErr{})
}

func (clt *Client) handleSessionNotFound(reqIdent [8]byte) {
	clt.requestManager.Fail(reqIdent, webwire.SessNotFoundErr{})
}

func (clt *Client) handleMaxSessConnsReached(reqIdent [8]byte) {
	clt.requestManager.Fail(reqIdent, webwire.MaxSessConnsReachedErr{})
}

func (clt *Client) handleSessionsDisabled(reqIdent [8]byte) {
	clt.requestManager.Fail(reqIdent, webwire.SessionsDisabledErr{})
}

func (clt *Client) handleReply(reqID [8]byte, payload webwire.Payload) {
	clt.requestManager.Fulfill(reqID, payload)
}

func (clt *Client) handleMessage(message []byte) error {
	if len(message) < 1 {
		return nil
	}
	switch message[0:1][0] {
	case webwire.MsgReplyBinary:
		clt.handleReply(
			extractMessageIdentifier(message),
			webwire.Payload{
				Encoding: webwire.EncodingBinary,
				Data:     message[9:],
			},
		)
	case webwire.MsgReplyUtf8:
		clt.handleReply(
			extractMessageIdentifier(message),
			webwire.Payload{
				Encoding: webwire.EncodingUtf8,
				Data:     message[9:],
			},
		)
	case webwire.MsgReplyUtf16:
		clt.handleReply(
			extractMessageIdentifier(message),
			webwire.Payload{
				Encoding: webwire.EncodingUtf16,
				Data:     message[10:],
			},
		)
	case webwire.MsgReplyShutdown:
		clt.handleReplyShutdown(extractMessageIdentifier(message))
	case webwire.MsgSessionNotFound:
		clt.handleSessionNotFound(extractMessageIdentifier(message))
	case webwire.MsgMaxSessConnsReached:
		clt.handleMaxSessConnsReached(extractMessageIdentifier(message))
	case webwire.MsgSessionsDisabled:
		clt.handleSessionsDisabled(extractMessageIdentifier(message))
	case webwire.MsgErrorReply:
		clt.handleFailure(extractMessageIdentifier(message), message[9:])
	case webwire.MsgReplyInternalError:
		clt.handleInternalError(extractMessageIdentifier(message))
	case webwire.MsgSignalBinary:
		clt.hooks.OnServerSignal(webwire.Payload{
			Encoding: webwire.EncodingBinary,
			Data:     message[2:],
		})
	case webwire.MsgSignalUtf8:
		clt.hooks.OnServerSignal(webwire.Payload{
			Encoding: webwire.EncodingUtf8,
			Data:     message[2:],
		})
	case webwire.MsgSignalUtf16:
		clt.hooks.OnServerSignal(webwire.Payload{
			Encoding: webwire.EncodingUtf16,
			Data:     message[2:],
		})
	case webwire.MsgSessionCreated:
		clt.handleSessionCreated(message[1:])
	case webwire.MsgSessionClosed:
		clt.handleSessionClosed()
	default:
		clt.warningLog.Printf(
			"Strange message type received: '%c'\n",
			message[0:1][0],
		)
	}
	return nil
}
