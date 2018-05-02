package conductor

import (
	"net/http"
)

// ConnectionAuthClient is the based interface for handling authenication for new HTTP requests that upgrade a websocket.
// Use this for checking for auth tokens and such to ensure only real clients can connect.
type ConnectionAuthClient interface {
	IsValid(r *http.Request) bool
}

// SimpleAuthClient is the default implmentation of ConnectionAuthClient.
// It simply accepts every websocket request.
// You probably shouldn't use this in production unless you want no auth for some reason.
type SimpleAuthClient struct {
}

// IsValid that accept every connection!
func (s *SimpleAuthClient) IsValid(r *http.Request) bool {
	// token := r.Header.Get(tokenKey)
	// params := r.URL.Query()
	// if len(params[tokenKey]) > 0 {
	// 	token = params[tokenKey][0]
	// }
	return true
}
