package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	proxymodifier "github.com/inference-gateway/inference-gateway/internal/proxy"

	gin "github.com/gin-gonic/gin"
	config "github.com/inference-gateway/inference-gateway/config"
	l "github.com/inference-gateway/inference-gateway/logger"
	providers "github.com/inference-gateway/inference-gateway/providers"
)

//go:generate mockgen -source=routes.go -destination=../tests/mocks/routes.go -package=mocks
type Router interface {
	ListModelsHandler(c *gin.Context)
	ChatCompletionsHandler(c *gin.Context)
	ProxyHandler(c *gin.Context)
	HealthcheckHandler(c *gin.Context)
	NotFoundHandler(c *gin.Context)
}

type RouterImpl struct {
	cfg      config.Config
	logger   l.Logger
	registry providers.ProviderRegistry
	client   providers.Client
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type ResponseJSON struct {
	Message string `json:"message"`
}

func NewRouter(cfg config.Config, logger l.Logger, registry providers.ProviderRegistry, client providers.Client) Router {
	return &RouterImpl{
		cfg,
		logger,
		registry,
		client,
	}
}

func (router *RouterImpl) NotFoundHandler(c *gin.Context) {
	router.logger.Error("requested route is not found", nil)
	c.JSON(http.StatusNotFound, ErrorResponse{Error: "Requested route is not found"})
}

func (router *RouterImpl) ProxyHandler(c *gin.Context) {
	p := providers.Provider(c.Param("provider"))
	provider, err := router.registry.BuildProvider(p, router.client)
	if err != nil {
		if strings.Contains(err.Error(), "token not configured") {
			router.logger.Error("provider requires authentication but no API key was configured", err, "provider", p)
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Provider requires an API key. Please configure the provider's API key."})
			return
		}
		router.logger.Error("provider not found or not supported", err, "provider", p)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Provider not found. Please check the list of supported providers."})
		return
	}

	// Setup authentication headers or query params
	token := provider.GetToken()
	switch provider.GetAuthType() {
	case providers.AuthTypeBearer:
		c.Request.Header.Set("Authorization", "Bearer "+token)
	case providers.AuthTypeXheader:
		c.Request.Header.Set("x-api-key", token)
	case providers.AuthTypeQuery:
		query := c.Request.URL.Query()
		query.Set("key", token)
		c.Request.URL.RawQuery = query.Encode()
	case providers.AuthTypeNone:
		// Do Nothing
	default:
		c.JSON(http.StatusUnprocessableEntity, ErrorResponse{Error: "Unsupported auth type"})
		return
	}

	// Add extra headers
	for key, values := range provider.GetExtraHeaders() {
		for _, value := range values {
			c.Request.Header.Add(key, value)
		}
	}

	// Check if streaming is requested
	isStreaming := c.Request.Header.Get("Accept") == "text/event-stream" || c.Request.Header.Get("Content-Type") == "text/event-stream"

	if isStreaming {
		handleStreamingRequest(c, provider, router)
		return
	}

	// Non-streaming case: Setup reverse proxy
	handleProxyRequest(c, provider, router)
}

func handleStreamingRequest(c *gin.Context, provider providers.IProvider, router *RouterImpl) {
	for k, v := range map[string]string{
		"Content-Type":      "text/event-stream",
		"Cache-Control":     "no-cache",
		"Connection":        "keep-alive",
		"Transfer-Encoding": "chunked",
	} {
		c.Header(k, v)
	}

	fullURL, err := constructProviderURL(provider, c.Param("path"), c.Request.URL.RawQuery)
	if err != nil {
		router.logger.Error("failed to construct provider URL", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Failed to construct URL"})
		return
	}

	// Read request body with a 10MB size limit for now, to prevent abuse
	// Will make it configurable later perhaps as a middleware
	const maxBodySize = 10 << 20
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxBodySize))
	if err != nil {
		router.logger.Error("failed to read request body", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Failed to read request"})
		return
	}
	if len(body) >= int(maxBodySize) {
		c.JSON(http.StatusRequestEntityTooLarge, ErrorResponse{Error: "Request body too large"})
		return
	}

	ctx := c.Request.Context()
	upstreamReq, err := http.NewRequestWithContext(ctx, c.Request.Method, fullURL.String(), bytes.NewReader(body))
	if err != nil {
		router.logger.Error("failed to create upstream request", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create upstream request"})
		return
	}

	upstreamReq.Header = c.Request.Header.Clone()

	resp, err := router.client.Do(upstreamReq)
	if err != nil {
		router.logger.Error("failed to make upstream request", err)
		c.JSON(http.StatusBadGateway, ErrorResponse{Error: "Failed to reach upstream server"})
		return
	}
	defer resp.Body.Close()

	reader := bufio.NewReaderSize(resp.Body, 4096)

	c.Stream(func(w io.Writer) bool {
		select {
		case <-ctx.Done():
			return false
		default:
		}

		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				router.logger.Error("failed to read stream", err,
					"url", fullURL.String(),
					"method", c.Request.Method)
			}
			return false
		}

		if len(line) == 0 {
			return true
		}

		router.logger.Debug("stream chunk",
			"provider", c.Param("provider"),
			"bytes", len(line),
			"data", string(bytes.TrimSpace(line)))

		if _, err := w.Write(line); err != nil {
			router.logger.Error("failed to write response", err,
				"bytes", len(line))
			return false
		}

		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}

		return true
	})
}

func handleProxyRequest(c *gin.Context, provider providers.IProvider, router *RouterImpl) {
	fullURL, err := constructProviderURL(provider, c.Param("path"), c.Request.URL.RawQuery)
	if err != nil {
		router.logger.Error("failed to construct provider URL", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Failed to construct URL"})
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(fullURL)

	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("Accept", "application/json")

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		router.logger.Error("proxy request failed", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		err = json.NewEncoder(w).Encode(ErrorResponse{
			Error: fmt.Sprintf("Failed to reach upstream server: %v", err),
		})
		if err != nil {
			router.logger.Error("failed to write error response", err)
		}
	}

	proxy.Director = func(req *http.Request) {
		req.Header = c.Request.Header
		req.Host = fullURL.Host
		req.URL.Host = fullURL.Host
		req.URL.Scheme = fullURL.Scheme
		req.URL.Path = fullURL.Path
		req.URL.RawQuery = fullURL.RawQuery

		if router.cfg.Environment == "development" {
			reqModifier := proxymodifier.NewDevRequestModifier(router.logger)
			if err := reqModifier.Modify(req); err != nil {
				router.logger.Error("failed to modify request", err)
				return
			}
		}
	}

	if router.cfg.Environment == "development" {
		devModifier := proxymodifier.NewDevResponseModifier(router.logger)
		proxy.ModifyResponse = devModifier.Modify
	}

	proxy.ServeHTTP(c.Writer, c.Request)
}

// constructProviderURL builds the provider URL consistently to avoid path duplication.
// It ensures that the path from the provider URL is handled correctly with the path parameter.
func constructProviderURL(provider providers.IProvider, pathParam, rawQuery string) (*url.URL, error) {
	providerURL, err := url.Parse(provider.GetURL())
	if err != nil {
		return nil, err
	}

	url := &url.URL{
		Scheme:   providerURL.Scheme,
		Host:     providerURL.Host,
		Path:     strings.TrimSuffix(providerURL.Path, "/") + "/" + strings.TrimPrefix(pathParam, "/"),
		RawQuery: rawQuery,
	}

	return url, nil
}

func (router *RouterImpl) HealthcheckHandler(c *gin.Context) {
	router.logger.Debug("healthcheck")
	c.JSON(http.StatusOK, ResponseJSON{Message: "OK"})
}

// ListModelsHandler implements an OpenAI-compatible API endpoint
// that returns model information in the standard OpenAI format.
//
// This handler supports the OpenAI GET /v1/models endpoint specification:
// https://platform.openai.com/docs/api-reference/models/list
//
// Parameters:
//   - provider (query): Optional. When specified, returns models from only that provider.
//     If not specified, returns models from all configured providers.
//
// Response format:
//
//	{
//	  "object": "list",
//	  "data": [
//	   {
//	      "id": "model-id",
//	      "object": "model",
//	      "created": 1686935002,
//	      "owned_by": "provider-name",
//	      "served_by": "provider-name"
//	   },
//	   ...
//	  ]
//	}
//
// This endpoint allows applications built for OpenAI's API to work seamlessly
// with the Inference Gateway's multi-provider architecture.
func (router *RouterImpl) ListModelsHandler(c *gin.Context) {
	providerID := providers.Provider(c.Query("provider"))
	if providerID != "" {
		provider, err := router.registry.BuildProvider(providerID, router.client)
		if err != nil {
			if strings.Contains(err.Error(), "token not configured") {
				router.logger.Error("provider requires authentication but no API key was configured", err, "provider", providerID)
				c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Provider requires an API key. Please configure the provider's API key."})
				return
			}
			router.logger.Error("provider not found or not supported", err, "provider", providerID)
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Provider not found. Please check the list of supported providers."})
			return
		}

		ctx, cancel := context.WithTimeout(c, router.cfg.Server.ReadTimeout)
		defer cancel()

		response, err := provider.ListModels(ctx)
		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				router.logger.Error("request timed out", err, "provider", provider.GetName())
				c.JSON(http.StatusGatewayTimeout, ErrorResponse{Error: "Request timed out"})
				return
			}
			router.logger.Error("failed to list models", err, "provider", provider.GetName())
			c.JSON(http.StatusBadGateway, ErrorResponse{Error: "Failed to list models"})
			return
		}

		c.JSON(http.StatusOK, response)
	} else {
		var wg sync.WaitGroup
		providersCfg := router.cfg.Providers

		ch := make(chan providers.ListModelsResponse, len(providersCfg))

		ctx, cancel := context.WithTimeout(c, router.cfg.Server.ReadTimeout*time.Millisecond)
		defer cancel()

		for providerID := range providersCfg {
			wg.Add(1)
			go func(id providers.Provider) {
				defer wg.Done()

				provider, err := router.registry.BuildProvider(id, router.client)
				if err != nil {
					router.logger.Error("failed to create provider", err, "provider", id)
					return
				}

				response, err := provider.ListModels(ctx)
				if err != nil {
					if ctx.Err() == context.DeadlineExceeded {
						router.logger.Error("request timed out", err, "provider", id)
						return
					}
					router.logger.Error("failed to list models", err, "provider", id)
					return
				}

				if response.Data == nil {
					response.Data = make([]providers.Model, 0)
				}
				ch <- response
			}(providerID)
		}

		wg.Wait()
		close(ch)

		var allModels []providers.Model
		for response := range ch {
			allModels = append(allModels, response.Data...)
		}

		if allModels == nil {
			allModels = make([]providers.Model, 0)
		}

		unifiedResponse := providers.ListModelsResponse{
			Object: "list",
			Data:   allModels,
		}

		c.JSON(http.StatusOK, unifiedResponse)
	}
}

// ChatCompletionsHandler implements an OpenAI-compatible API endpoint
// that generates text completions in the standard OpenAI format.
//
// Regular response format:
//
//	{
//	  "choices": [
//	    {
//	      "finish_reason": "stop",
//	      "message": {
//	        "content": "Hello, how can I help you today?",
//	        "role": "assistant"
//	      }
//	    }
//	  ],
//	  "created": 1742165657,
//	  "id": "chatcmpl-118",
//	  "model": "deepseek-r1:1.5b",
//	  "object": "chat.completion",
//	  "usage": {
//	    "completion_tokens": 139,
//	    "prompt_tokens": 10,
//	    "total_tokens": 149
//	  }
//	}
//
// Streaming response format:
//
//	{
//	  "choices": [
//	    {
//	      "index": 0,
//	      "finish_reason": "stop",
//	      "delta": {
//	        "content": "Hello",
//	        "role": "assistant"
//	      }
//	    }
//	  ],
//	  "created": 1742165657,
//	  "id": "chatcmpl-118",
//	  "model": "deepseek-r1:1.5b",
//	  "object": "chat.completion.chunk",
//	  "usage": {
//	    "completion_tokens": 139,
//	    "prompt_tokens": 10,
//	    "total_tokens": 149
//	  }
//	}
//
// It returns token completions as chat in the standard OpenAI format, allowing applications
// built for OpenAI's API to work seamlessly with the Inference Gateway's multi-provider
// architecture.
func (router *RouterImpl) ChatCompletionsHandler(c *gin.Context) {
	var req providers.CreateChatCompletionRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		router.logger.Error("failed to decode request", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Failed to decode request"})
		return
	}

	model := req.Model
	providerID := providers.Provider(c.Query("provider"))
	if providerID == "" {
		var providerPtr *providers.Provider
		providerPtr, model = determineProviderAndModelName(model)
		if providerPtr == nil {
			router.logger.Error("unable to determine provider for model", nil, "model", req.Model)
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Unable to determine provider for model. Please specify a provider."})
			return
		}
		providerID = *providerPtr
	}
	req.Model = model

	provider, err := router.registry.BuildProvider(providerID, router.client)
	if err != nil {
		if strings.Contains(err.Error(), "token not configured") {
			router.logger.Error("provider requires authentication but no API key was configured", err, "provider", providerID)
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Provider requires an API key. Please configure the provider's API key."})
			return
		}
		router.logger.Error("provider not found or not supported", err, "provider", providerID)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Provider not found. Please check the list of supported providers."})
		return
	}

	router.logger.Debug("server read timeout", "timeout", router.cfg.Server.ReadTimeout)

	ctx, cancel := context.WithTimeout(c, router.cfg.Server.ReadTimeout)
	defer cancel()

	// Streaming response
	if req.Stream != nil && *req.Stream {
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("Transfer-Encoding", "chunked")

		streamCh, err := provider.StreamChatCompletions(ctx, req)
		if err != nil {
			router.logger.Error("failed to start streaming", err)
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Failed to start streaming"})
			return
		}

		c.Stream(func(w io.Writer) bool {
			select {
			case line, ok := <-streamCh:
				if !ok {
					if _, err := c.Writer.Write([]byte("data: [DONE]\n\n")); err != nil {
						router.logger.Error("failed to write [DONE] marker", err)
					}
					return false
				}

				if _, err := c.Writer.Write([]byte("data: " + string(line) + "\n\n")); err != nil {
					router.logger.Error("failed to write chunk", err)
					return false
				}
				c.Writer.Flush()
				return true
			case <-ctx.Done():
				return false
			}
		})
		return
	}

	// Non-streaming response
	response, err := provider.ChatCompletions(ctx, req)
	if err != nil {
		if err == context.DeadlineExceeded || ctx.Err() == context.DeadlineExceeded {
			router.logger.Error("request timed out", err, "provider", providerID)
			c.JSON(http.StatusGatewayTimeout, ErrorResponse{Error: "Request timed out"})
			return
		}
		router.logger.Error("failed to generate tokens", err, "provider", providerID)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: fmt.Sprintf("Failed to generate tokens: %s", err)})
		return
	}

	c.JSON(http.StatusOK, response)
}

func determineProviderAndModelName(model string) (provider *providers.Provider, modelName string) {
	modelLower := strings.ToLower(model)

	// First check for explicit provider prefixes (ollama-, groq-, etc.)
	providerPrefixMapping := map[string]providers.Provider{
		"ollama/":     providers.OllamaID,
		"groq/":       providers.GroqID,
		"cloudflare/": providers.CloudflareID,
		"openai/":     providers.OpenaiID,
		"anthropic/":  providers.AnthropicID,
		"cohere/":     providers.CohereID,
		"deepseek/":   providers.DeepseekID,
	}

	for prefix, providerID := range providerPrefixMapping {
		if strings.HasPrefix(modelLower, prefix) {
			return &providerID, strings.TrimPrefix(model, prefix)
		}
	}

	// Then check for model-name based prefixes (gpt-, claude-, etc.)
	modelPrefixMapping := map[string]providers.Provider{
		"gpt-":      providers.OpenaiID,
		"claude-":   providers.AnthropicID,
		"llama-":    providers.GroqID,
		"command-":  providers.CohereID,
		"deepseek-": providers.GroqID,
	}

	for prefix, providerID := range modelPrefixMapping {
		if strings.HasPrefix(modelLower, prefix) {
			return &providerID, model
		}
	}

	return nil, model
}
