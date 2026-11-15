package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/ATLURI0001/llm-hallucination-detector/internal/claim"
	"github.com/ATLURI0001/llm-hallucination-detector/internal/engine"
	"github.com/ATLURI0001/llm-hallucination-detector/internal/models"
)

// Handler manages the REST API endpoints.
type Handler struct {
	startTime time.Time
	version   string
}

// NewHandler initializes a new API handler.
func NewHandler(version string) *Handler {
	return &Handler{
		startTime: time.Now(),
		version:   version,
	}
}

// HandleHealth handles GET /api/v1/health
func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	resp := HealthResponse{
		Status:    "healthy",
		Version:   h.version,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Uptime:    time.Since(h.startTime).String(),
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleVerify handles POST /api/v1/verify
func (h *Handler) HandleVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	var req VerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON request body: "+err.Error())
		return
	}

	if strings.TrimSpace(req.AnswerText) == "" {
		writeError(w, http.StatusBadRequest, "Field 'answer_text' is required")
		return
	}
	if strings.TrimSpace(req.SourceText) == "" {
		writeError(w, http.StatusBadRequest, "Field 'source_text' is required")
		return
	}

	opts := models.DefaultVerificationOptions()
	if req.Options != nil {
		opts = *req.Options
	}

	detector := engine.NewDetector(opts)
	result, err := detector.Verify(r.Context(), req.AnswerText, req.SourceText)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Verification engine error: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// HandleExtractClaims handles POST /api/v1/extract
func (h *Handler) HandleExtractClaims(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	var req ExtractClaimsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON request body: "+err.Error())
		return
	}

	if strings.TrimSpace(req.Text) == "" {
		writeError(w, http.StatusBadRequest, "Field 'text' is required")
		return
	}

	if req.MinClaimLength < 0 {
		req.MinClaimLength = 0
	}

	extractor := claim.NewExtractor(req.MinClaimLength)
	claims := extractor.ExtractClaims(req.Text)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total_claims": len(claims),
		"claims":       claims,
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Error: msg,
		Code:  status,
	})
}
