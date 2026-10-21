package tests

import (
	"testing"

	"github.com/asepcepot6140-glitch/netcore/internal/utils"
)

func TestGenerateAccessTokenRequiresStrongSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "too-short")

	if _, err := utils.GenerateAccessToken(1); err == nil {
		t.Fatal("GenerateAccessToken() error = nil, want weak secret error")
	}
}

func TestGenerateAndParseToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-that-is-at-least-32-bytes")

	token, err := utils.GenerateAccessToken(42)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	claims, err := utils.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}
	if claims.UserID != 42 || claims.Type != "access" {
		t.Fatalf("claims = %+v, want user ID 42 and access type", claims)
	}
}
