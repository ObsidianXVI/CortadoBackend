package auth

import "time"

const (
	DefaultAPIKeysCollection         = "api_keys"
	DefaultFirstPartyUsersCollection = "users"
	DefaultRefreshTokensCollection   = "refresh_tokens"

	ActorTypePlatform = "platform"
	ActorTypeUser     = "user"

	APIKeyKindPersonal = "personal"
	APIKeyKindPlatform = "platform"
)

type APIKeyRecord struct {
	CreatedAt       time.Time `firestore:"createdAt"`
	CreatedByUserID string    `firestore:"createdByUserId,omitempty"`
	Hash            string    `firestore:"hash"`
	ID              string    `firestore:"id"`
	Kind            string    `firestore:"kind,omitempty"`
	Revoked         bool      `firestore:"revoked"`
	TenantID        string    `firestore:"tenantId"`
	UserID          string    `firestore:"userId,omitempty"`
}

type RefreshTokenRecord struct {
	CreatedAt    time.Time `firestore:"createdAt"`
	ExpiresAt    time.Time `firestore:"expiresAt"`
	ActorType    string    `firestore:"actorType,omitempty"`
	JTI          string    `firestore:"jti"`
	RefreshToken string    `firestore:"refreshToken"`
	TenantID     string    `firestore:"tenantId"`
	UserID       string    `firestore:"userId"`
}

type SessionTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type FirstPartyAccount struct {
	CreatedAt        time.Time `firestore:"createdAt"`
	DisplayName      string    `firestore:"displayName,omitempty"`
	Email            string    `firestore:"email,omitempty"`
	FirebaseUID      string    `firestore:"firebaseUid"`
	PersonalTenantID string    `firestore:"personalTenantId"`
	UpdatedAt        time.Time `firestore:"updatedAt"`
	UserID           string    `firestore:"userId"`
}

type PersonalTenantRecord struct {
	CreatedAt   time.Time `firestore:"createdAt"`
	DisplayName string    `firestore:"displayName,omitempty"`
	Kind        string    `firestore:"kind"`
	OwnerUserID string    `firestore:"ownerUserId"`
	TenantID    string    `firestore:"tenantId"`
	UpdatedAt   time.Time `firestore:"updatedAt"`
}

type PlatformTenantRecord struct {
	CreatedAt   time.Time `firestore:"createdAt"`
	DisplayName string    `firestore:"displayName,omitempty"`
	Kind        string    `firestore:"kind"`
	OwnerUserID string    `firestore:"ownerUserId"`
	TenantID    string    `firestore:"tenantId"`
	UpdatedAt   time.Time `firestore:"updatedAt"`
}

type APIKey struct {
	CreatedAt time.Time `json:"createdAt"`
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Revoked   bool      `json:"revoked"`
	TenantID  string    `json:"tenantId"`
	UserID    string    `json:"userId,omitempty"`
}

type IssuedAPIKey struct {
	APIKey string `json:"apiKey"`
	Record APIKey `json:"record"`
}

type APIKeyIdentity struct {
	Kind     string `json:"kind"`
	TenantID string `json:"tenantId"`
	UserID   string `json:"userId,omitempty"`
}

type PlatformTenant struct {
	CreatedAt   time.Time `json:"createdAt"`
	DisplayName string    `json:"displayName,omitempty"`
	Kind        string    `json:"kind"`
	TenantID    string    `json:"tenantId"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (r APIKeyRecord) Metadata() APIKey {
	return APIKey{
		CreatedAt: r.CreatedAt,
		ID:        r.ID,
		Kind:      normalizeAPIKeyKind(r.Kind),
		Revoked:   r.Revoked,
		TenantID:  r.TenantID,
		UserID:    r.UserID,
	}
}

func (r PlatformTenantRecord) Metadata() PlatformTenant {
	return PlatformTenant{
		CreatedAt:   r.CreatedAt,
		DisplayName: r.DisplayName,
		Kind:        r.Kind,
		TenantID:    r.TenantID,
		UpdatedAt:   r.UpdatedAt,
	}
}
