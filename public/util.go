package public

import (
	"crypto/sha256"
	"encoding/hex"
)

func GenSaltPassword(salt string, password string) string {
	h := sha256.New()
	h.Write([]byte(salt + password))
	return hex.EncodeToString(h.Sum(nil))
}
