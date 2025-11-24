package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

func newTestAuthHandler(t *testing.T) (*AuthHandlers, sqlmock.Sqlmock) {
	t.Helper()
	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	db := sqlx.NewDb(rawDB, "sqlmock")
	logger := zerolog.Nop()
	handler := NewAuthHandlers(db, logger, "mock", "anon", "test-secret")
	return handler, mock
}

func TestLoginMockModeCreatesUserAndReturnsToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, mock := newTestAuthHandler(t)

	email := "newuser@example.com"
	password := "password123"

	mock.ExpectQuery(`SELECT id FROM user_profiles WHERE email = \$1`).
		WithArgs(email).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(`INSERT INTO user_profiles`).
		WithArgs(sqlmock.AnyArg(), email, email, "user").
		WillReturnResult(sqlmock.NewResult(1, 1))

	body, _ := json.Marshal(LoginRequest{Email: email, Password: password})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	h.Login(c)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if resp["token"] == nil || resp["access_token"] == nil {
		t.Fatalf("expected token fields in response, got %v", resp)
	}
}

func TestGetCurrentUserReturnsProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, mock := newTestAuthHandler(t)

	userID := "00000000-0000-0000-0000-000000000001"
	now := time.Now()
	profileRows := sqlmock.NewRows([]string{"id", "email", "full_name", "role", "is_active", "created_at", "updated_at"}).
		AddRow(userID, "admin@example.com", "Admin", "admin", true, now, now)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, email, full_name, role, is_active, created_at, updated_at
		FROM user_profiles
		WHERE id = $1
	`)).
		WithArgs(userID).
		WillReturnRows(profileRows)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("userID", userID)

	h.GetCurrentUser(c)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var resp UserProfileResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if resp.User.ID != userID || resp.User.Email != "admin@example.com" {
		t.Fatalf("unexpected profile: %+v", resp.User)
	}
}
