package conductor

import (
	"crypto/sha1"
	"fmt"
	"testing"
)

func TestHashToken(t *testing.T) {
	token := "token"
	data := fmt.Sprintf("%x", sha1.Sum([]byte(token)))
	hash := hashToken(token)
	if data != hash {
		t.Error("token is not being properly hashed.")
	}
}
