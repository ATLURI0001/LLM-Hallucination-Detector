package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// LLMJudge coordinates with an external LLM (Gemini or OpenAI API) for semantic entailment.
type LLMJudge struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

// NewLLMJudge creates a new LLM judge client.
func NewLLMJudge(apiKey, model, baseURL string) *LLMJudge {
	if model == "" {
		model = "gemini-2.5-flash"
	}
	if baseURL == "" {
		// Default to Gemini REST endpoint if gemini model, or OpenAI compatible
		if strings.HasPrefix(model, "gemini") {
			baseURL = "https://generativelanguage.googleapis.com/v1beta/models"
		} else {
			baseURL = "https://api.openai.com/v1"
		}
	}
	return &LLMJudge{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

// JudgeEntailmentRequest is the JSON schema payload for verification.
type LLMJudgeResponse struct {
	Status      ClaimStatus       `json:"status"`
	Type        HallucinationType `json:"hallucination_type"`
	Confidence  float64           `json:"confidence"`
	Explanation string            `json:"explanation"`
}

// EvaluateEntailment calls the configured LLM API to evaluate premise vs hypothesis.
func (j *LLMJudge) EvaluateEntailment(ctx context.Context, premise, hypothesis string) (*LLMJudgeResponse, error) {
	if j.apiKey == "" {
		return nil, fmt.Errorf("no LLM API key provided")
	}

	prompt := fmt.Sprintf(`You are an expert Natural Language Inference (NLI) and fact-checking judge.
Determine if the Hypothesis is supported by, contradicted by, or unsupported by the Premise.

Premise (Source Document):
"%s"

Hypothesis (Claim to Verify):
"%s"

Return ONLY a valid JSON object matching this schema:
{
  "status": "SUPPORTED" | "CONTRADICTED" | "UNSUPPORTED" | "PARTIALLY_SUPPORTED",
  "hallucination_type": "NONE" | "EXTRINSIC" | "INTRINSIC" | "NUMERIC_MISMATCH" | "ENTITY_CONFUSION",
  "confidence": 0.0 to 1.0,
  "explanation": "concise explanation of whether premise entails, contradicts, or fails to support hypothesis"
}`, premise, hypothesis)

	if strings.HasPrefix(j.model, "gemini") {
		return j.callGemini(ctx, prompt)
	}
	return j.callOpenAI(ctx, prompt)
}

func (j *LLMJudge) callGemini(ctx context.Context, prompt string) (*LLMJudgeResponse, error) {
	url := fmt.Sprintf("%s/%s:generateContent?key=%s", j.baseURL, j.model, j.apiKey)

	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature":      0.0,
			"responseMimeType": "application/json",
		},
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := j.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gemini api error (HTTP %d): %s", resp.StatusCode, string(b))
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from gemini")
	}

	rawJSON := result.Candidates[0].Content.Parts[0].Text
	var judgeResp LLMJudgeResponse
	if err := json.Unmarshal([]byte(rawJSON), &judgeResp); err != nil {
		return nil, fmt.Errorf("failed to parse judge json: %w", err)
	}

	return &judgeResp, nil
}

func (j *LLMJudge) callOpenAI(ctx context.Context, prompt string) (*LLMJudgeResponse, error) {
	url := fmt.Sprintf("%s/chat/completions", j.baseURL)

	reqBody := map[string]interface{}{
		"model": j.model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are a precise Natural Language Inference fact verifier. Always respond in JSON."},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.0,
		"response_format": map[string]string{
			"type": "json_object",
		},
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+j.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := j.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("llm api error (HTTP %d): %s", resp.StatusCode, string(b))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("empty response from llm")
	}

	var judgeResp LLMJudgeResponse
	if err := json.Unmarshal([]byte(result.Choices[0].Message.Content), &judgeResp); err != nil {
		return nil, fmt.Errorf("failed to parse llm json: %w", err)
	}

	return &judgeResp, nil
}
