package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	firebase "firebase.google.com/go/v4"
	firebaseauth "firebase.google.com/go/v4/auth"
)

type VerifiedFirebaseToken struct {
	Claims map[string]any
	UID    string
}

type FirebaseTokenVerifier interface {
	VerifyIDToken(ctx context.Context, idToken string) (*VerifiedFirebaseToken, error)
}

type FirebaseClaimsManager interface {
	SetCustomUserClaims(ctx context.Context, uid string, claims map[string]any) error
}

type FirebaseVerifier struct {
	client *firebaseauth.Client
}

func NewFirebaseVerifier(ctx context.Context, projectID string) (*FirebaseVerifier, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, errors.New("firebase project id is required")
	}

	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: strings.TrimSpace(projectID)})
	if err != nil {
		return nil, fmt.Errorf("initialize firebase app: %w", err)
	}
	client, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("initialize firebase auth client: %w", err)
	}

	return &FirebaseVerifier{client: client}, nil
}

func (v *FirebaseVerifier) VerifyIDToken(ctx context.Context, idToken string) (*VerifiedFirebaseToken, error) {
	if strings.TrimSpace(idToken) == "" {
		return nil, ErrFirebaseTokenMissing
	}

	token, err := v.client.VerifyIDToken(ctx, idToken)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFirebaseTokenInvalid, err)
	}
	if strings.TrimSpace(token.UID) == "" {
		return nil, ErrFirebaseTokenInvalid
	}

	return &VerifiedFirebaseToken{
		Claims: token.Claims,
		UID:    token.UID,
	}, nil
}

func (v *FirebaseVerifier) SetCustomUserClaims(ctx context.Context, uid string, claims map[string]any) error {
	if strings.TrimSpace(uid) == "" {
		return ErrFirebaseTokenInvalid
	}
	if err := v.client.SetCustomUserClaims(ctx, strings.TrimSpace(uid), claims); err != nil {
		return fmt.Errorf("set firebase custom claims: %w", err)
	}
	return nil
}

func TenantIDFromFirebaseClaims(claims map[string]any, claimKey string) (string, error) {
	key := strings.TrimSpace(claimKey)
	if key == "" {
		key = "tenant_id"
	}

	value, ok := claims[key]
	if !ok {
		return "", ErrTenantClaimMissing
	}
	tenantID, ok := value.(string)
	if !ok || strings.TrimSpace(tenantID) == "" {
		return "", ErrTenantClaimMissing
	}
	return strings.TrimSpace(tenantID), nil
}

func MergeFirebaseClaims(existing map[string]any, updates map[string]any) map[string]any {
	merged := make(map[string]any, len(existing)+len(updates))
	for key, value := range existing {
		merged[key] = value
	}
	for key, value := range updates {
		merged[key] = value
	}
	return merged
}
