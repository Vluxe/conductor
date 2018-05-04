package conductor

import (
	"net/http"
)

// ConnectionAuth is the based interface for handling authentication and authorization.
// This is used for new HTTP requests that upgrade a websocket and the permissions to channels.
// Use this for checking for auth tokens and such to ensure only real clients can connect.
// and that connections have the right permissions to write or bind to a channel.
type ConnectionAuth interface {
	IsValid(r *http.Request) bool                    // IsValid is called on every HTTP upgrade request and should be used to validate auth tokens.
	ConnToRequest(r *http.Request, conn Connection)  // ConnToRequest is called so you can map your HTTP request (like the auth token) to a current connection.
	CanBind(conn Connection, message *Message) bool  // CanBind is called on every bind request, so optimizing it is highly recommended.
	CanWrite(conn Connection, message *Message) bool // CanWrite is called on every write request, so optimizing it is highly recommended.
}

// SimpleAuth is the default implmentation of ConnectionAuth.
// It simply accepts every websocket request.
// You probably shouldn't use this in production unless you want no auth for some reason.
type SimpleAuth struct {
}

// NewSimpleAuth creates a SimpleAuth to use.
// Don't use this for anything but testing as you need to create proper auth
func NewSimpleAuth() *SimpleAuth {
	return &SimpleAuth{}
}

// IsValid that accept every connection!
func (s *SimpleAuth) IsValid(r *http.Request) bool {
	// token := r.Header.Get(tokenKey)
	// params := r.URL.Query()
	// if len(params[tokenKey]) > 0 {
	// 	token = params[tokenKey][0]
	// }
	return true
}

// ConnToRequest does nothing because simple auth doesn't use auth tokens!
func (s *SimpleAuth) ConnToRequest(r *http.Request, conn Connection) {
	//noop
}

// CanBind should check if a connection has rights to bind to channel.
// It accepts any client that request to bind.
func (s *SimpleAuth) CanBind(conn Connection, message *Message) bool {
	return true
}

// CanWrite should check if a connection has rights to write to channel.
// This makes sure the connection is bound to channel before allowing a write.
func (s *SimpleAuth) CanWrite(conn Connection, message *Message) bool {
	for _, channel := range conn.Channels() {
		if channel == message.ChannelName {
			return true
		}
	}
	return false
}
