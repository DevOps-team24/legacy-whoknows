package auth

import (
	"crypto/md5"
	"encoding/hex"
)

// HashPassword produces an MD5 hex digest identical to Python's
// hashlib.md5(password.encode('utf-8')).hexdigest(), keeping
// backwards-compatibility with the legacy database.
func HashPassword(password string) string {
	h := md5.Sum([]byte(password))
	return hex.EncodeToString(h[:])
}

func VerifyPassword(storedHash, password string) bool {
	return storedHash == HashPassword(password)
}
