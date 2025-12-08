//go:build ignore

package main

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func main() {
	secret := "cqvZfI6pA9A7OZOQh6gDRcvd2/LBC7U1550HY965sGrd59hlPo6qq/sqLFLeryHomjuHNnYZKfso1heHgS/Kng=="

	claims := jwt.MapClaims{
		"sub":   "00000000-0000-0000-0000-000000000001",
		"email": "admin@test.com",
		"role":  "admin",
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		panic(err)
	}
	fmt.Println(tokenString)
}
