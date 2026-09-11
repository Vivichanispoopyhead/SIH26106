package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
)

type aiSettingsRequest struct {
	Provider string `json:"provider"`
	APIKey   string `json:"api_key"`
	Model    string `json:"model"`
}

func (h emailHandler) getAISettings(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.service.AIProviderStatus())
}

func (h emailHandler) updateAISettings(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	var request aiSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "AI settings must be valid JSON.")
		return
	}
	if provider := strings.ToLower(strings.TrimSpace(request.Provider)); provider != "" && provider != "google" && provider != "gemini" {
		writeError(w, http.StatusBadRequest, "UNSUPPORTED_AI_PROVIDER", "Only Google Gemini is currently supported.")
		return
	}
	if strings.TrimSpace(request.APIKey) == "" {
		writeError(w, http.StatusBadRequest, "AI_API_KEY_REQUIRED", "An AI provider API key is required.")
		return
	}
	if len(request.APIKey) > 4096 {
		writeError(w, http.StatusBadRequest, "AI_API_KEY_INVALID", "The AI provider API key is too long.")
		return
	}
	if err := h.service.ConfigureAI(request.APIKey, request.Model); err != nil {
		writeError(w, http.StatusBadRequest, "AI_CONFIGURATION_INVALID", "The AI provider configuration is invalid.")
		return
	}
	writeJSON(w, http.StatusOK, h.service.AIProviderStatus())
}
