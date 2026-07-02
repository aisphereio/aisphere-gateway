package server

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	v1 "aisphere-gateway/api/gateway/v1"
	"aisphere-gateway/internal/conf"
	"aisphere-gateway/internal/data"
	"aisphere-gateway/internal/service"
	"github.com/aisphereio/kernel/gatewayx"
	"github.com/aisphereio/kernel/logx"
	"github.com/aisphereio/kernel/metricsx"
	khttp "github.com/aisphereio/kernel/transportx/http"
)

func NewHTTPServer(cfg conf.ServerConfig, logCfg logx.Config, metricsCfg conf.MetricsConfig, logger logx.Logger, metrics metricsx.Manager, resources *data.Resources, admin *service.GatewayAdminService, dispatcher gatewayx.Dispatcher) *khttp.Server {
	addr := cfg.HTTP.Addr
	if addr == "" {
		addr = "0.0.0.0:8000"
	}
	timeout := cfg.HTTP.Timeout
	if timeout <= 0 {
		timeout = time.Second
	}
	opts := []khttp.ServerOption{
		khttp.Address(addr),
		khttp.Timeout(timeout),
		khttp.Logger(logger.Named("transport.http")),
		khttp.AccessLog(logCfg.AccessLog),
		khttp.CORS(cfg.HTTP.CORS),
	}
	if metricsCfg.Enabled {
		opts = append(opts, khttp.Metrics(metrics))
	}
	srv := khttp.NewServer(opts...)
	v1.RegisterGatewayAdminServiceHTTPServer(srv, admin)
	srv.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	srv.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if resources == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	srv.HandlePrefix("/v1/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		headers := make(map[string]string, len(r.Header))
		for key, values := range r.Header {
			if len(values) > 0 {
				if skipDispatchHeader(key) {
					continue
				}
				headers[key] = values[0]
			}
		}
		if r.URL.RawQuery != "" {
			headers["X-Gateway-Raw-Query"] = r.URL.RawQuery
		}
		resp, err := dispatcher.Dispatch(r.Context(), gatewayx.DispatchRequest{
			Method:  r.Method,
			Path:    r.URL.Path,
			Headers: headers,
			Body:    body,
		})
		if err != nil {
			status := resp.Status
			if status == 0 {
				status = http.StatusBadGateway
			}
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}
		status := resp.Status
		if status == 0 {
			status = http.StatusOK
		}
		writeJSON(w, status, resp.Body)
	}))
	return srv
}

func skipDispatchHeader(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "connection",
		"keep-alive",
		"proxy-authenticate",
		"proxy-authorization",
		"te",
		"trailer",
		"transfer-encoding",
		"upgrade",
		"content-length":
		return true
	default:
		return false
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
