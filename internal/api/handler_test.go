package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ATLURI0001/llm-hallucination-detector/internal/api"
	"github.com/ATLURI0001/llm-hallucination-detector/internal/models"
)

func TestHandleHealth(t *testing.T) {
	handler := api.NewHandler("1.0.0")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()

	handler.HandleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp api.HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "healthy" {
		t.Errorf("expected status 'healthy', got %q", resp.Status)
	}
	if resp.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", resp.Version)
	}
	if resp.Timestamp == "" || resp.Uptime == "" {
		t.Errorf("expected non-empty timestamp and uptime")
	}
}

func TestHandleVerify(t *testing.T) {
	handler := api.NewHandler("1.0.0")

	t.Run("MethodNotAllowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/verify", nil)
		w := httptest.NewRecorder()
		handler.HandleVerify(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", w.Code)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/verify", bytes.NewBufferString("invalid json"))
		w := httptest.NewRecorder()
		handler.HandleVerify(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("MissingAnswerText", func(t *testing.T) {
		body, _ := json.Marshal(api.VerifyRequest{
			AnswerText: "",
			SourceText: "Some source context here.",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/verify", bytes.NewReader(body))
		w := httptest.NewRecorder()
		handler.HandleVerify(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("MissingSourceText", func(t *testing.T) {
		body, _ := json.Marshal(api.VerifyRequest{
			AnswerText: "The sky is blue.",
			SourceText: "",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/verify", bytes.NewReader(body))
		w := httptest.NewRecorder()
		handler.HandleVerify(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("SuccessfulVerification", func(t *testing.T) {
		body, _ := json.Marshal(api.VerifyRequest{
			AnswerText: "Apple reported $90 billion revenue.",
			SourceText: "Apple Inc. reported quarterly revenue of $90 billion today.",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/verify", bytes.NewReader(body))
		w := httptest.NewRecorder()
		handler.HandleVerify(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		var result models.VerificationResult
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode result: %v", err)
		}
		if result.TotalClaims == 0 {
			t.Errorf("expected claims to be extracted, got 0")
		}
	})
}

func TestHandleExtractClaims(t *testing.T) {
	handler := api.NewHandler("1.0.0")

	t.Run("MethodNotAllowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/extract", nil)
		w := httptest.NewRecorder()
		handler.HandleExtractClaims(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", w.Code)
		}
	})

	t.Run("MissingText", func(t *testing.T) {
		body, _ := json.Marshal(api.ExtractClaimsRequest{
			Text: "",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/extract", bytes.NewReader(body))
		w := httptest.NewRecorder()
		handler.HandleExtractClaims(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("SuccessfulExtraction", func(t *testing.T) {
		body, _ := json.Marshal(api.ExtractClaimsRequest{
			Text: "Google was founded in 1998 by Larry Page and Sergey Brin.",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/extract", bytes.NewReader(body))
		w := httptest.NewRecorder()
		handler.HandleExtractClaims(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if total, ok := resp["total_claims"].(float64); !ok || total == 0 {
			t.Errorf("expected total_claims > 0, got %v", resp["total_claims"])
		}
	})
}
