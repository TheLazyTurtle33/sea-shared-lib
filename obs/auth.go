package obs

import (
	"crypto/sha256"
	"encoding/base64"
)

// buildAuth constructs the authentication string required by OBS WebSocket v5.
// Algorithm: base64( sha256( password + salt ) ) then base64( sha256( that + challenge ) )
func buildAuth(password, salt, challenge string) (string, error) {
	h1 := sha256.Sum256([]byte(password + salt))
	b1 := base64.StdEncoding.EncodeToString(h1[:])

	h2 := sha256.Sum256([]byte(b1 + challenge))
	return base64.StdEncoding.EncodeToString(h2[:]), nil
}
