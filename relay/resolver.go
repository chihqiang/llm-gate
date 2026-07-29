package relay

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"chihqiang/llm-gate/model"

	"gorm.io/gorm"
)

type ResolveResult struct {
	Model    *model.ModelConfig
	Provider *model.Provider
}

type chatRequest struct {
	Model string `json:"model"`
}

func ResolveModel(r *http.Request, db *gorm.DB) (*ResolveResult, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, errors.New("failed to read request body")
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

	var req chatRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, errors.New("invalid request body")
	}

	if req.Model == "" {
		return nil, errors.New("model is required")
	}

	var mc model.ModelConfig
	if err := db.Where("name = ? AND status = ?", req.Model, true).
		Order("id ASC").
		First(&mc).Error; err != nil {
		return nil, errors.New("model not found or disabled")
	}

	var provider model.Provider
	if err := db.Where("id = ? AND status = ?", mc.ProviderID, true).First(&provider).Error; err != nil {
		return nil, errors.New("provider not found or disabled")
	}

	return &ResolveResult{Model: &mc, Provider: &provider}, nil
}
