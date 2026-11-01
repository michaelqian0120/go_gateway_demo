package public

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt"
	"golang_common/lib"
)

func jwtSignKey() ([]byte, error) {
	secret := lib.GetStringConf("proxy.jwt.secret")
	if secret == "" {
		// 拒绝用空密钥签发或校验，否则任何人都能伪造token
		return nil, errors.New("proxy.jwt.secret not configured")
	}
	return []byte(secret), nil
}

// JwtEncode 用 HS256 签发租户token
func JwtEncode(claims jwt.StandardClaims) (string, error) {
	key, err := jwtSignKey()
	if err != nil {
		return "", err
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(key)
}

// JwtDecode 校验签名和有效期，返回token中的声明
func JwtDecode(tokenString string) (*jwt.StandardClaims, error) {
	key, err := jwtSignKey()
	if err != nil {
		return nil, err
	}
	claims := &jwt.StandardClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// 只接受HMAC签名，防止把 alg 改成 none 等方式伪造token
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return key, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
