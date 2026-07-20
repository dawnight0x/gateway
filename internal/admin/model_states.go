package admin

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
)

type modelStateResetInput struct {
	ProviderID string `json:"providerId"`
	KeyID      string `json:"keyId"`
	ModelID    string `json:"modelId"`
}

func (s *Service) modelStates(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) != 2 || parts[1] != "reset" {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var input modelStateResetInput
	if !decodeJSONBody(w, r, &input) {
		return
	}
	input.ProviderID = strings.TrimSpace(input.ProviderID)
	input.KeyID = strings.TrimSpace(input.KeyID)
	input.ModelID = strings.TrimSpace(input.ModelID)
	if input.ProviderID == "" || input.ModelID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "providerId and modelId are required"})
		return
	}
	var err error
	if input.KeyID == "" {
		err = s.store.ResetProviderModelStates(r.Context(), input.ProviderID, input.ModelID)
	} else {
		err = s.store.ResetProviderModelState(r.Context(), input.ProviderID, input.KeyID, input.ModelID)
	}
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
