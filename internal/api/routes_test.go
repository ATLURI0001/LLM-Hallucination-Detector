package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/ATLURI0001/llm-hallucination-detector/internal/api"
)

func TestSetupRouter(t *testing.T) {
	handler := api.NewHandler("1.0.0")
	mockFS := fstest.MapFS{
		"style.css": &fstest.MapFile{
			Data: []byte("body { color: red; }"),
		},
	}
	router := api.SetupRouter(handler, mockFS)

	t.Run("CORSPreflight", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/api/v1/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 for OPTIONS, got %d", w.Code)
		}
		if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
			t.Errorf("expected Access-Control-Allow-Origin '*', got %q", origin)
		}
	})

	t.Run("NotFoundOnApiPrefix", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/nonexistent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404 for unknown api endpoint, got %d", w.Code)
		}
	})

	t.Run("StaticFileServing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/style.css", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 for static file, got %d", w.Code)
		}
		if w.Body.String() != "body { color: red; }" {
			t.Errorf("unexpected body: %s", w.Body.String())
		}
	})
}
