package connectproxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"code-code.internal/go-contract/platform/model/v1/modelservicev1connect"
	"code-code.internal/go-contract/platform/provider/v1/providerservicev1connect"
)

// ConsolePathPrefix is the browser-facing base path for proxied Connect RPCs.
const ConsolePathPrefix = "/api/connect"

// Config groups platform Connect upstreams exposed through the console proxy.
type Config struct {
	ModelBaseURL    string
	ProviderBaseURL string
}

// NewHandler returns the console-facing Connect proxy for allowlisted platform RPCs.
func NewHandler(config Config) (http.Handler, error) {
	modelTarget, err := parseBaseURL(config.ModelBaseURL)
	if err != nil {
		return nil, fmt.Errorf("connectproxy: model upstream: %w", err)
	}
	providerTarget, err := parseBaseURL(config.ProviderBaseURL)
	if err != nil {
		return nil, fmt.Errorf("connectproxy: provider upstream: %w", err)
	}
	return newHandler(map[string]*url.URL{
		modelservicev1connect.ModelServiceListModelsProcedure:        modelTarget,
		providerservicev1connect.ProviderServiceListVendorsProcedure: providerTarget,
	}), nil
}

type handler struct {
	proxies map[string]*httputil.ReverseProxy
}

func newHandler(targets map[string]*url.URL) http.Handler {
	proxies := make(map[string]*httputil.ReverseProxy, len(targets))
	for procedure, target := range targets {
		proxies[procedure] = newProxy(target)
	}
	return &handler{proxies: proxies}
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	procedure := strings.TrimPrefix(r.URL.Path, ConsolePathPrefix)
	if procedure == r.URL.Path {
		http.NotFound(w, r)
		return
	}
	proxy, ok := h.proxies[procedure]
	if !ok {
		http.NotFound(w, r)
		return
	}
	proxy.ServeHTTP(w, r)
}

func newProxy(target *url.URL) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Director: func(request *http.Request) {
			request.URL.Scheme = target.Scheme
			request.URL.Host = target.Host
			request.URL.Path = joinPath(target.Path, strings.TrimPrefix(request.URL.Path, ConsolePathPrefix))
			request.URL.RawPath = ""
			request.URL.RawQuery = joinQuery(target.RawQuery, request.URL.RawQuery)
			request.Host = target.Host
		},
	}
}

func parseBaseURL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("connectproxy: base url is empty")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("connectproxy: parse base url: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("connectproxy: base url must use http or https")
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("connectproxy: base url host is empty")
	}
	return parsed, nil
}

func joinPath(basePath, requestPath string) string {
	switch {
	case basePath == "" || basePath == "/":
		return requestPath
	case requestPath == "":
		return basePath
	case strings.HasSuffix(basePath, "/") && strings.HasPrefix(requestPath, "/"):
		return basePath + strings.TrimPrefix(requestPath, "/")
	case !strings.HasSuffix(basePath, "/") && !strings.HasPrefix(requestPath, "/"):
		return basePath + "/" + requestPath
	default:
		return basePath + requestPath
	}
}

func joinQuery(baseQuery, requestQuery string) string {
	switch {
	case baseQuery == "":
		return requestQuery
	case requestQuery == "":
		return baseQuery
	default:
		return baseQuery + "&" + requestQuery
	}
}
