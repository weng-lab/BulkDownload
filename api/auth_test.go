package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdminAuth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		configToken string
		headerToken string
		wantStatus  int
	}{
		{
			name:       "blank configured token rejects",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:        "missing request header rejects",
			configToken: "secret-token",
			wantStatus:  http.StatusUnauthorized,
		},
		{
			name:        "wrong request header rejects",
			configToken: "secret-token",
			headerToken: "wrong-token",
			wantStatus:  http.StatusUnauthorized,
		},
		{
			name:        "matching request header allows request",
			configToken: "secret-token",
			headerToken: "secret-token",
			wantStatus:  http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			called := false
			handler := adminAuth(tt.configToken)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusNoContent)
			}))

			req := httptest.NewRequest(http.MethodGet, "/admin/jobs", nil)
			if tt.headerToken != "" {
				req.Header.Set("X-Admin-Token", tt.headerToken)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantStatus == http.StatusNoContent && !called {
				t.Fatal("expected next handler to be called")
			}
			if tt.wantStatus != http.StatusNoContent && called {
				t.Fatal("expected next handler not to be called")
			}
		})
	}
}
