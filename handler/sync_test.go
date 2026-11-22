package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestSyncRejectsInvalidStrategy(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/sync", strings.NewReader(`{"target_ids":[1],"strategy":"bogus"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	h := NewSyncHandler(nil, "")
	if err := h.Sync(e.NewContext(req, rec)); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "invalid strategy") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}
