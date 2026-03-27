package utils

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func SignToken(id, username, role string) (string, error) {
	jwtSecret := os.Getenv("JWT_SECRET")
	jwtExpires := os.Getenv("JWT_EXPIRES")

	claims := jwt.MapClaims{
		"uid":   id,
		"uname": username,
		"role":  role,
	}

	if jwtExpires != "" {
		duration, err := time.ParseDuration(jwtExpires)
		if err != nil {
			fmt.Println(err)
			return "", ErrorHandler(err, "Internal sever error")
		}
		claims["exp"] = jwt.NewNumericDate(time.Now().Add(duration))
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		fmt.Println(err)
		return "", ErrorHandler(err, "Internal server error")
	}

	return signedToken, nil
}

type JWTStore struct {
	mu       sync.Mutex
	storeMap map[string]time.Time
}

var JwtStore = JWTStore{
	storeMap: make(map[string]time.Time),
}

func (store *JWTStore) AddTokenToMap(token string, timeStamp time.Time) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.storeMap[token] = timeStamp
}

func (store *JWTStore) DeleteExpiredTokensBG() {
	for {
		time.Sleep(2 * time.Minute)
		store.mu.Lock()
		for token, expirationTime := range store.storeMap {
			if time.Now().After(expirationTime) {
				delete(store.storeMap, token)
			}
		}
		store.mu.Unlock()
	}
}

func (store *JWTStore) IsLoggedOut(token string) bool {
	store.mu.Lock()
	defer store.mu.Unlock()
	_, ok := store.storeMap[token]
	return ok
}
