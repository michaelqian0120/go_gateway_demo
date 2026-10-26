package public

import (
	"crypto/sha256"
	"encoding/hex"
)

func GenSaltPassword(salt string, password string) string {
	s1 := sha256.New()
	s1.Write([]byte(password))
	str1 := hex.EncodeToString(s1.Sum(nil))
	s2 := sha256.New()
	s2.Write([]byte(str1 + salt))
	return hex.EncodeToString(s2.Sum(nil))
}
