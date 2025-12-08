package errors

import (
	"errors"
	"testing"
)

func TestLocalizedMessage_Portuguese(t *testing.T) {
	err := New("ERR_TEST", "English message", "Mensagem em portugues", 400)

	tests := []struct {
		lang     string
		expected string
	}{
		{"pt", "Mensagem em portugues"},
		{"pt-BR", "Mensagem em portugues"},
		{"pt_BR", "Mensagem em portugues"},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			if got := err.LocalizedMessage(tt.lang); got != tt.expected {
				t.Errorf("LocalizedMessage(%q) = %q, want %q", tt.lang, got, tt.expected)
			}
		})
	}
}

func TestLocalizedMessage_English(t *testing.T) {
	err := New("ERR_TEST", "English message", "Mensagem em portugues", 400)

	tests := []struct {
		lang     string
		expected string
	}{
		{"en", "English message"},
		{"en-US", "English message"},
		{"", "English message"},
		{"fr", "English message"}, // Unknown defaults to English
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			if got := err.LocalizedMessage(tt.lang); got != tt.expected {
				t.Errorf("LocalizedMessage(%q) = %q, want %q", tt.lang, got, tt.expected)
			}
		})
	}
}

func TestWithInternal(t *testing.T) {
	baseErr := New("ERR_TEST", "Test error", "Erro de teste", 500)
	internalErr := errors.New("database connection failed")

	wrapped := baseErr.WithInternal(internalErr)

	// Check fields are preserved
	if wrapped.Code != baseErr.Code {
		t.Errorf("Code = %q, want %q", wrapped.Code, baseErr.Code)
	}
	if wrapped.MessageEN != baseErr.MessageEN {
		t.Errorf("MessageEN = %q, want %q", wrapped.MessageEN, baseErr.MessageEN)
	}
	if wrapped.MessagePT != baseErr.MessagePT {
		t.Errorf("MessagePT = %q, want %q", wrapped.MessagePT, baseErr.MessagePT)
	}
	if wrapped.HTTPStatus != baseErr.HTTPStatus {
		t.Errorf("HTTPStatus = %d, want %d", wrapped.HTTPStatus, baseErr.HTTPStatus)
	}

	// Check internal error is set
	if wrapped.Internal != internalErr {
		t.Error("Internal error not set correctly")
	}

	// Check Unwrap works
	if wrapped.Unwrap() != internalErr {
		t.Error("Unwrap() did not return internal error")
	}
}

func TestWithDetail(t *testing.T) {
	baseErr := New("ERR_VALIDATION", "Validation failed", "Falha na validacao", 400)

	detailed := baseErr.WithDetail("field", "email").WithDetail("reason", "invalid format")

	if detailed.Details["field"] != "email" {
		t.Errorf("Details[field] = %q, want %q", detailed.Details["field"], "email")
	}
	if detailed.Details["reason"] != "invalid format" {
		t.Errorf("Details[reason] = %q, want %q", detailed.Details["reason"], "invalid format")
	}

	// Ensure original error is not modified
	if baseErr.Details != nil && len(baseErr.Details) > 0 {
		t.Error("Original error was modified")
	}
}

func TestError(t *testing.T) {
	err := New("ERR_TEST", "English error message", "Mensagem de erro", 400)

	if err.Error() != "English error message" {
		t.Errorf("Error() = %q, want %q", err.Error(), "English error message")
	}
}

func TestIsAppError(t *testing.T) {
	appErr := New("ERR_TEST", "Test", "Teste", 400)
	stdErr := errors.New("standard error")

	if !IsAppError(appErr) {
		t.Error("IsAppError(appErr) = false, want true")
	}
	if IsAppError(stdErr) {
		t.Error("IsAppError(stdErr) = true, want false")
	}
	if IsAppError(nil) {
		t.Error("IsAppError(nil) = true, want false")
	}
}

func TestAsAppError(t *testing.T) {
	appErr := New("ERR_TEST", "Test", "Teste", 400)
	stdErr := errors.New("standard error")

	if got, ok := AsAppError(appErr); !ok || got.Code != "ERR_TEST" {
		t.Error("AsAppError failed for AppError")
	}

	if _, ok := AsAppError(stdErr); ok {
		t.Error("AsAppError should return false for standard error")
	}
}

func TestWrap(t *testing.T) {
	baseErr := New("ERR_DB", "Database error", "Erro no banco de dados", 500)
	internalErr := errors.New("connection timeout")

	wrapped := Wrap(baseErr, internalErr)

	if wrapped.Internal != internalErr {
		t.Error("Wrap did not set internal error")
	}
	if wrapped.Code != baseErr.Code {
		t.Error("Wrap did not preserve code")
	}
}
