package auth

import "time"

const (
	DefaultAPIKeysCollection       = "api_keys"
	DefaultRefreshTokensCollection = "refresh_tokens"
)

type APIKeyRecord struct {
	ID        string    `firestore:"id"`
	Hash      string    `firestore:"hash"`
	Revoked   bool      `firestore:"revoked"`
	TenantID  string    `firestore:"tenantId"`
	CreatedAt time.Time `firestore:"createdAt"`
}

type RefreshTokenRecord struct {
	CreatedAt    time.Time `firestore:"createdAt"`
	ExpiresAt    time.Time `firestore:"expiresAt"`
	JTI          string    `firestore:"jti"`
	RefreshToken string    `firestore:"refreshToken"`
	TenantID     string    `firestore:"tenantId"`
	UserID       string    `firestore:"userId"`
}

type SessionTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}
