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
	UserID    string    `firestore:"userId"`
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

type APIKey struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenantId"`
	UserID    string    `json:"userId"`
	Revoked   bool      `json:"revoked"`
	CreatedAt time.Time `json:"createdAt"`
}

type IssuedAPIKey struct {
	APIKey string `json:"apiKey"`
	Record APIKey `json:"record"`
}

type APIKeyIdentity struct {
	TenantID string `json:"tenantId"`
	UserID   string `json:"userId,omitempty"`
}

func (r APIKeyRecord) Metadata() APIKey {
	return APIKey{
		ID:        r.ID,
		TenantID:  r.TenantID,
		UserID:    r.UserID,
		Revoked:   r.Revoked,
		CreatedAt: r.CreatedAt,
	}
}
