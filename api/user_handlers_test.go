package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

func newTestUserHandler(t *testing.T) (*UserHandlers, sqlmock.Sqlmock) {
	t.Helper()
	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	db := sqlx.NewDb(rawDB, "sqlmock")
	logger := zerolog.Nop()
	return NewUserHandlers(db, logger), mock
}

func TestDeleteAccountSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, mock := newTestUserHandler(t)

	userID := "00000000-0000-0000-0000-000000000001"
	mock.ExpectExec(`UPDATE user_profiles`).
		WithArgs(userID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/user", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("userID", userID)

	handler.DeleteAccount(c)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

func TestDeleteAccountUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _ := newTestUserHandler(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/user", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.DeleteAccount(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

func TestDeleteAccountNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, mock := newTestUserHandler(t)

	userID := "00000000-0000-0000-0000-000000000099"
	mock.ExpectExec(`UPDATE user_profiles`).
		WithArgs(userID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/user", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("userID", userID)

	handler.DeleteAccount(c)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}
