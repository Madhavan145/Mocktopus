package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenManager struct {
	secret          []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

type Claims struct {
	UserID  string `json:"uid"`
	OrgID   string `json:"org"`
	Role    string `json:"role"`
	IsAdmin bool   `json:"adm,omitempty"`
	jwt.RegisteredClaims
}

func NewTokenManager(secret string, accessTokenTTL, refreshTokenTTL time.Duration) TokenManager {
	return TokenManager{
		secret:          []byte(secret),
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
	}
}

func (tm TokenManager) CreateTokenPair(userID, orgID, role string, isAdmin bool) (accessToken string, refreshToken string, refreshExpiry time.Time, err error) {
	now := time.Now().UTC()
	accessExpiry := now.Add(tm.accessTokenTTL)
	refreshExpiry = now.Add(tm.refreshTokenTTL)

	claims := Claims{
		UserID:  userID,
		OrgID:   orgID,
		Role:    role,
		IsAdmin: isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(accessExpiry),
		},
	}

	access := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err = access.SignedString(tm.secret)
	if err != nil {
		return "", "", time.Time{}, err
	}

	refreshToken = uuid.NewString()
	return accessToken, refreshToken, refreshExpiry, nil
}

func (tm TokenManager) ParseAccessToken(accessToken string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(accessToken, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return tm.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, jwt.ErrTokenMalformed
	}
	return claims, nil
}

func HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
