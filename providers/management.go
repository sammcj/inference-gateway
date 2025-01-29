package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	l "github.com/inference-gateway/inference-gateway/logger"
)

//go:generate mockgen -source=management.go -destination=../tests/mocks/provider.go -package=mocks
type Provider interface {
	// Getters
	GetID() string
	GetName() string
	GetURL() string
	GetToken() string
	GetAuthType() string
	GetExtraHeaders() map[string][]string
	GetClient() Client

	// Fetchers
	ListModels(ctx context.Context) (ListModelsResponse, error)
	GenerateTokens(ctx context.Context, model string, messages []Message) (GenerateResponse, error)
}

type ProviderImpl struct {
	id           string
	name         string
	url          string
	token        string
	authType     string
	extraHeaders map[string][]string
	endpoints    Endpoints
	client       Client
	logger       l.Logger
}

func NewProvider(cfg map[string]*Config, id string, logger *l.Logger, client *Client) (Provider, error) {
	provider, ok := cfg[id]
	if !ok {
		return nil, fmt.Errorf("provider %s not found", id)
	}

	if provider.AuthType != AuthTypeNone && provider.Token == "" {
		return nil, fmt.Errorf("provider %s token not configured", id)
	}

	return &ProviderImpl{
		id:           provider.ID,
		name:         provider.Name,
		url:          provider.URL,
		token:        provider.Token,
		authType:     provider.AuthType,
		extraHeaders: provider.ExtraHeaders,
		endpoints:    provider.Endpoints,
		client:       *client,
		logger:       *logger,
	}, nil
}

func (p *ProviderImpl) GetID() string {
	return p.id
}

func (p *ProviderImpl) GetName() string {
	return p.name
}

func (p *ProviderImpl) GetURL() string {
	baseURL := p.url
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}
	return baseURL
}

func (p *ProviderImpl) GetToken() string {
	return p.token
}

func (p *ProviderImpl) GetAuthType() string {
	return p.authType
}

func (p *ProviderImpl) GetExtraHeaders() map[string][]string {
	return p.extraHeaders
}

func (p *ProviderImpl) EndpointList() string {
	return p.endpoints.List
}

func (p *ProviderImpl) EndpointGenerate() string {
	return p.endpoints.Generate
}

func (p *ProviderImpl) SetClient(client Client) {
	p.client = client
}

func (p *ProviderImpl) GetClient() Client {
	return p.client
}

func (p *ProviderImpl) ListModels(ctx context.Context) (ListModelsResponse, error) {
	baseURL, err := url.Parse(p.GetURL())
	if err != nil {
		p.logger.Error("failed to parse base URL", err)
		return ListModelsResponse{}, fmt.Errorf("failed to parse base URL: %v", err)
	}

	url := "/proxy/" + p.GetID() + baseURL.Path + p.EndpointList()

	p.logger.Debug("list models", "url", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		p.logger.Error("failed to create request", err)
		return ListModelsResponse{}, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		p.logger.Error("failed to make request", err, "provider", p.GetName())
		return ListModelsResponse{
			Provider: p.GetID(),
			Models:   make([]Model, 0),
		}, fmt.Errorf("failed to reach provider %s: %w", p.GetName(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ListModelsResponse{
			Provider: p.GetID(),
			Models:   make([]Model, 0),
		}, fmt.Errorf("failed with status code: %d", resp.StatusCode)
	}

	switch p.GetID() {
	case OllamaID:
		var response ListModelsResponseOllama
		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			p.logger.Error("failed to decode response", err, "provider", p.GetName())
			return ListModelsResponse{}, fmt.Errorf("failed to decode response: %w", err)
		}
		return response.Transform(), nil
	case GroqID:
		var response ListModelsResponseGroq
		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			p.logger.Error("failed to decode response", err, "provider", p.GetName())
			return ListModelsResponse{}, fmt.Errorf("failed to decode response: %w", err)
		}
		return response.Transform(), nil
	case OpenaiID:
		var response ListModelsResponseOpenai
		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			p.logger.Error("failed to decode response", err, "provider", p.GetName())
			return ListModelsResponse{}, fmt.Errorf("failed to decode response: %w", err)
		}
		return response.Transform(), nil
	case GoogleID:
		var response ListModelsResponseGoogle
		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			p.logger.Error("failed to decode response", err, "provider", p.GetName())
			return ListModelsResponse{}, fmt.Errorf("failed to decode response: %w", err)
		}
		return response.Transform(), nil
	case CloudflareID:
		var response ListModelsResponseCloudflare
		if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
			p.logger.Error("failed to decode response", err, "provider", p.GetName())
			return ListModelsResponse{}, fmt.Errorf("failed to decode response: %w", err)
		}
		return response.Transform(), nil
	case CohereID:
		var response ListModelsResponseCohere
		if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
			p.logger.Error("failed to decode response", err, "provider", p.GetName())
			return ListModelsResponse{}, fmt.Errorf("failed to decode response: %w", err)
		}
		return response.Transform(), nil
	case AnthropicID:
		var response ListModelsResponseAnthropic
		if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
			p.logger.Error("failed to decode response", err, "provider", p.GetName())
			return ListModelsResponse{}, fmt.Errorf("failed to decode response: %w", err)
		}
		return response.Transform(), nil
	default:
		p.logger.Error("provider not found", nil, "provider", p.GetName())
		return ListModelsResponse{}, fmt.Errorf("failed to decode response: %w", err)
	}
}

func (p *ProviderImpl) GenerateTokens(ctx context.Context, model string, messages []Message) (GenerateResponse, error) {
	if p == nil {
		return GenerateResponse{}, errors.New("provider cannot be nil")
	}

	baseURL, err := url.Parse(p.GetURL())
	if err != nil {
		p.logger.Error("failed to parse base URL", err)
		return GenerateResponse{}, fmt.Errorf("failed to parse base URL: %v", err)
	}

	// Construct URL with model parameter if needed
	url := "/proxy/" + p.GetID() + baseURL.Path + p.EndpointGenerate()
	if p.GetID() == GoogleID || p.GetID() == CloudflareID {
		url = strings.Replace(url, "{model}", model, 1)
	}

	genRequest := GenerateRequest{
		Model:    model,
		Messages: messages,
	}

	switch p.GetID() {
	case OllamaID:
		// Request
		payload := genRequest.TransformOllama()
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			p.logger.Error("failed to marshal request", err)
			return GenerateResponse{}, fmt.Errorf("failed to marshal request: %w", err)
		}

		resp, err := fetchTokens(ctx, p.client, url, payloadBytes, p.logger)
		if err != nil {
			p.logger.Error("failed to make request", err)
			return GenerateResponse{}, fmt.Errorf("failed to make request: %w", err)
		}

		// Response
		var response GenerateResponseOllama
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			p.logger.Error("failed to decode response", err)
			return GenerateResponse{}, fmt.Errorf("failed to decode response: %w", err)
		}
		return response.Transform(), nil
	case GroqID:
		// Request
		payload := genRequest.TransformGroq()
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			p.logger.Error("failed to marshal request", err)
			return GenerateResponse{}, fmt.Errorf("failed to marshal request: %w", err)
		}

		resp, err := fetchTokens(ctx, p.client, url, payloadBytes, p.logger)
		if err != nil {
			p.logger.Error("failed to make request", err)
			return GenerateResponse{}, fmt.Errorf("failed to make request: %w", err)
		}

		// Response
		var response GenerateResponseGroq
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			p.logger.Error("failed to decode response", err)
			return GenerateResponse{}, fmt.Errorf("failed to decode response: %w", err)
		}
		return response.Transform(), nil
	case OpenaiID:
		// Request
		payload := genRequest.TransformOpenai()
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			p.logger.Error("failed to marshal request", err)
			return GenerateResponse{}, fmt.Errorf("failed to marshal request: %w", err)
		}

		resp, err := fetchTokens(ctx, p.client, url, payloadBytes, p.logger)
		if err != nil {
			p.logger.Error("failed to make request", err)
			return GenerateResponse{}, fmt.Errorf("failed to make request: %w", err)
		}

		// Response
		var response GenerateResponseOpenai
		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			p.logger.Error("failed to decode response", err, "provider", p.GetName())
			return GenerateResponse{}, fmt.Errorf("failed to decode response: %w", err)
		}
		return response.Transform(), nil
	case GoogleID:
		// Request
		payload := genRequest.TransformGoogle()
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			p.logger.Error("failed to marshal request", err)
			return GenerateResponse{}, fmt.Errorf("failed to marshal request: %w", err)
		}

		resp, err := fetchTokens(ctx, p.client, url, payloadBytes, p.logger)
		if err != nil {
			p.logger.Error("failed to make request", err)
			return GenerateResponse{}, fmt.Errorf("failed to make request: %w", err)
		}

		// Response
		var response GenerateResponseGoogle
		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			p.logger.Error("failed to decode response", err, "provider", p.GetName())
			return GenerateResponse{}, fmt.Errorf("failed to decode response: %w", err)
		}
		return response.Transform(), nil
	case CloudflareID:
		// Request
		payload := genRequest.TransformCloudflare()
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			p.logger.Error("failed to marshal request", err)
			return GenerateResponse{}, fmt.Errorf("failed to marshal request: %w", err)
		}

		resp, err := fetchTokens(ctx, p.client, url, payloadBytes, p.logger)
		if err != nil {
			p.logger.Error("failed to make request", err)
			return GenerateResponse{}, fmt.Errorf("failed to make request: %w", err)
		}

		// Response
		var response GenerateResponseCloudflare
		if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
			p.logger.Error("failed to decode response", err, "provider", p.GetName())
			return GenerateResponse{}, fmt.Errorf("failed to decode response: %w", err)
		}
		return response.Transform(), nil
	case CohereID:
		// Request
		payload := genRequest.TransformCohere()
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			p.logger.Error("failed to marshal request", err)
			return GenerateResponse{}, fmt.Errorf("failed to marshal request: %w", err)
		}

		resp, err := fetchTokens(ctx, p.client, url, payloadBytes, p.logger)
		if err != nil {
			p.logger.Error("failed to make request", err)
			return GenerateResponse{}, fmt.Errorf("failed to make request: %w", err)
		}

		// Response
		var response GenerateResponseCohere
		if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
			p.logger.Error("failed to decode response", err, "provider", p.GetName())
			return GenerateResponse{}, fmt.Errorf("failed to decode response: %w", err)
		}
		return response.Transform(), nil
	case AnthropicID:
		// Request
		payload := genRequest.TransformAnthropic()
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			p.logger.Error("failed to marshal request", err)
			return GenerateResponse{}, fmt.Errorf("failed to marshal request: %w", err)
		}

		resp, err := fetchTokens(ctx, p.client, url, payloadBytes, p.logger)
		if err != nil {
			p.logger.Error("failed to make request", err)
			return GenerateResponse{}, fmt.Errorf("failed to make request: %w", err)
		}

		// Response
		var response GenerateResponseAnthropic
		if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
			p.logger.Error("failed to decode response", err, "provider", p.GetName())
			return GenerateResponse{}, fmt.Errorf("failed to decode response: %w", err)
		}
		return response.Transform(), nil
	default:
		p.logger.Error("unsupported provider", nil)
		return GenerateResponse{}, fmt.Errorf("unsupported provider: %s", p.GetID())
	}
}

func fetchTokens(ctx context.Context, client Client, url string, payload []byte, logger l.Logger) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(payload)))
	if err != nil {
		logger.Error("failed to create request", err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		logger.Error("failed to make request", err)
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.Error("request failed", fmt.Errorf("status code: %d", resp.StatusCode))
		return nil, fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	return resp, nil
}
