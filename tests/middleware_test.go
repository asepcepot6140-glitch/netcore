package tests

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/asepcepot6140-glitch/netcore/internal/middleware"
	"github.com/asepcepot6140-glitch/netcore/internal/utils"

	"github.com/gin-gonic/gin"
)

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("JWT_SECRET", "test-secret-that-is-at-least-32-bytes")

	accessToken, err := utils.GenerateAccessToken(42)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	refreshToken, err := utils.GenerateRefreshToken(42)
	if err != nil {
		t.Fatalf("GenerateRefreshToken() error = %v", err)
	}

	tests := []struct {
		name          string
		authorization string
		wantStatus    int
		wantBody      string
	}{
		{
			name:       "missing token",
			wantStatus: http.StatusUnauthorized,
			wantBody:   "{\"message\":\"Token autentikasi diperlukan\"}",
		},
		{
			name:          "refresh token",
			authorization: "Bearer " + refreshToken,
			wantStatus:    http.StatusUnauthorized,
			wantBody:      "{\"message\":\"Token autentikasi tidak valid\"}",
		},
		{
			name:          "valid access token",
			authorization: "Bearer " + accessToken,
			wantStatus:    http.StatusOK,
			wantBody:      "42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(middleware.Auth())
			router.GET("/", func(c *gin.Context) {
				c.String(http.StatusOK, strconv.FormatUint(uint64(c.GetUint("user_id")), 10))
			})

			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.Header.Set("Authorization", tt.authorization)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, tt.wantStatus)
			}
			if response.Body.String() != tt.wantBody {
				t.Fatalf("body = %q, want %q", response.Body.String(), tt.wantBody)
			}
		})
	}
}
