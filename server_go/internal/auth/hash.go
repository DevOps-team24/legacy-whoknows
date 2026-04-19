package auth

import (
	"crypto/md5" // #nosec G501 -- Required for legacy database compatibility; hashes must match existing stored MD5 values.
	"encoding/hex"
)

// HashPassword produces an MD5 hex digest identical to Python's
// hashlib.md5(password.encode('utf-8')).hexdigest(), keeping
// backwards-compatibility with the legacy database.
func HashPassword(password string) string {
	// #nosec G401 -- Required for legacy database compatibility; changing the algorithm would invalidate existing credentials.
	h := md5.Sum([]byte(password))
	return hex.EncodeToString(h[:])
}

func VerifyPassword(storedHash, password string) bool {
	return storedHash == HashPassword(password)
}
