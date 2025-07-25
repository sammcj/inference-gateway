package providers

import (
	"strings"
	"time"
)

type ModelCohere struct {
	Name             string   `json:"name,omitempty"`
	Endpoints        []string `json:"endpoints,omitempty"`
	Finetuned        bool     `json:"finetuned,omitempty"`
	ContextLenght    int32    `json:"context_length,omitempty"`
	TokenizerURL     string   `json:"tokenizer_url,omitempty"`
	SupportsVision   bool     `json:"supports_vision,omitempty"`
	DefaultEndpoints []string `json:"default_endpoints,omitempty"`
}

type ListModelsResponseCohere struct {
	NextPageToken string         `json:"next_page_token,omitempty"`
	Models        []*ModelCohere `json:"models,omitempty"`
}

func (l *ListModelsResponseCohere) Transform() ListModelsResponse {
	provider := CohereID
	models := make([]Model, len(l.Models))
	created := time.Now().Unix()
	for i, model := range l.Models {
		modelID := model.Name
		if !strings.Contains(modelID, "/") {
			modelID = string(provider) + "/" + modelID
		}
		models[i] = Model{
			ID:       modelID,
			Object:   "model",
			Created:  created,
			OwnedBy:  string(provider),
			ServedBy: provider,
		}
	}

	return ListModelsResponse{
		Provider: &provider,
		Object:   "list",
		Data:     models,
	}
}
