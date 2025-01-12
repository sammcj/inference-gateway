package gateway

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	logger "github.com/edenreich/inference-gateway/logger"
	otel "github.com/edenreich/inference-gateway/otel"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	trace "go.opentelemetry.io/otel/trace"
)

func Create(target string, apiKey string, prefix string, tp otel.TracerProvider, enableTelemetry bool, logger logger.Logger) http.HandlerFunc {
	parsedTarget, err := url.Parse(target)
	if err != nil {
		logger.Error("Failed to parse target URL", err)
		return nil
	}

	proxy := httputil.NewSingleHostReverseProxy(parsedTarget)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		logger.Error("Proxy error", err)
		http.Error(w, "Proxy error: "+err.Error(), http.StatusBadGateway)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if enableTelemetry {
			ctx := r.Context()
			_, span := tp.Tracer("inference-gateway").Start(ctx, "proxy-request")
			defer span.End()
			span.AddEvent("Proxying request", trace.WithAttributes(
				semconv.HTTPMethodKey.String(r.Method),
				semconv.HTTPTargetKey.String(r.URL.String()),
				semconv.HTTPRequestContentLengthKey.Int64(r.ContentLength),
			))
		}

		if apiKey == "" && prefix != "/llms/ollama/" {
			http.Error(w, "API token is not configured. Please set it by environment variable.", http.StatusUnauthorized)
			return
		}
		r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
		proxy.Director = func(req *http.Request) {
			if prefix != "/llms/ollama/" && prefix != "/llms/google/" {
				req.Header.Set("Authorization", "Bearer "+apiKey)
			}
			if prefix == "/llms/google/" {
				query := req.URL.Query()
				query.Set("key", apiKey)
				req.URL.RawQuery = query.Encode()
			}

			req.Header.Set("Accept", "application/json")
			req.Header.Set("Content-Type", "application/json")
			req.URL.Scheme = parsedTarget.Scheme
			req.URL.Host = parsedTarget.Host
			req.URL.Path = strings.TrimRight(parsedTarget.Path, "/") + "/" + strings.TrimLeft(r.URL.Path, "/")
			req.Host = parsedTarget.Host

			logger.Info("Proxying request", "method", req.Method, "target_url", req.URL.String())
		}
		proxy.ServeHTTP(w, r)
	}
}
