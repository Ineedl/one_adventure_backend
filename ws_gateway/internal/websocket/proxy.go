package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gogf/gf/v2/net/ghttp"
	clientv3 "go.etcd.io/etcd/client/v3"
	obslog "one_adventure_observability_log"
	"one_adventure_servicekit/api-contract/gateway_server_discovery"
	"one_adventure_servicekit/token"
)

type registrationInfo struct {
	Address string `json:"address"`
	WSPort  int    `json:"ws_port"`
}

type Proxy struct{ etcd *clientv3.Client }

var verifier *token.RS256Verifier

func SetVerifier(v *token.RS256Verifier) { verifier = v }

func NewProxy(endpoints []string) (*Proxy, error) {
	client, err := clientv3.New(clientv3.Config{Endpoints: endpoints, DialTimeout: time.Second})
	if err != nil {
		return nil, err
	}
	return &Proxy{etcd: client}, nil
}

func (p *Proxy) Close() error { return p.etcd.Close() }

// Handle adapts the standard HTTP proxy to GoFrame's handler signature.
func (p *Proxy) Handle(request *ghttp.Request) {
	p.ServeHTTP(request.Response.Writer, request.Request)
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	obslog.Info(r.Context(), "websocket proxy request", map[string]any{"path": r.URL.Path, "remote_addr": r.RemoteAddr})
	value := r.URL.Query().Get("token")
	if value == "" {
		value = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	}
	if verifier == nil || value == "" {
		obslog.Warn(r.Context(), "websocket proxy authentication rejected")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if _, err := verifier.VerifyAndParse(value); err != nil {
		obslog.Warn(r.Context(), "websocket proxy token invalid", map[string]any{"error": err.Error()})
		http.Error(w, "invalid or expired token", http.StatusUnauthorized)
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 3 || parts[0] != "ws_gateway" || parts[2] != "ws" {
		http.NotFound(w, r)
		return
	}
	serverID, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil || serverID == 0 {
		http.Error(w, "invalid server_id", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	resp, err := p.etcd.Get(ctx, gateway_server_discovery.ServerKey(serverID))
	if err != nil {
		http.Error(w, "resolve gate server failed", http.StatusBadGateway)
		return
	}
	if len(resp.Kvs) == 0 {
		obslog.Warn(r.Context(), "gate server registration not found", map[string]any{"server_id": serverID})
		http.Error(w, "gate server is offline", http.StatusServiceUnavailable)
		return
	}
	var info registrationInfo
	if json.Unmarshal(resp.Kvs[0].Value, &info) != nil || info.Address == "" || info.WSPort < 1 {
		http.Error(w, "invalid gate server registration", http.StatusBadGateway)
		return
	}
	target, _ := url.Parse(fmt.Sprintf("http://%s:%d", info.Address, info.WSPort))
	proxy := httputil.NewSingleHostReverseProxy(target)
	original := proxy.Director
	proxy.Director = func(req *http.Request) { original(req); req.URL.Path = "/ws" }
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, _ error) {
		obslog.Error(r.Context(), "gate server websocket proxy failed", map[string]any{"server_id": serverID})
		http.Error(w, "gate server unavailable", http.StatusBadGateway)
	}
	proxy.ServeHTTP(w, r)
}
