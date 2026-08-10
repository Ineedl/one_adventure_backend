package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/gogf/gf/v2/net/ghttp"
	"one_adventure_servicekit/httpresponse"
	servicetoken "one_adventure_servicekit/token"
)

type Middleware struct {
	verifier  servicetoken.Verifier
	whitelist []string
}

func New(ctx context.Context) (*Middleware, error) {
	config, err := loadConfig(ctx)
	if err != nil {
		return nil, err
	}
	verifier, err := servicetoken.NewRS256VerifierFromFile(config.PublicKeyPath, config.Issuer)
	if err != nil {
		return nil, err
	}
	return &Middleware{verifier: verifier, whitelist: config.Whitelist}, nil
}

func (m *Middleware) Handle(request *ghttp.Request) {
	if m.isWhitelisted(request.URL.Path) {
		request.Middleware.Next()
		return
	}
	value, ok := bearerToken(request.Header.Get("Authorization"))
	if !ok {
		m.unauthorized(request)
		return
	}
	claims, err := m.verifier.VerifyAndParse(value)
	if err != nil {
		m.unauthorized(request)
		return
	}
	request.SetCtx(servicetoken.WithUserInfo(request.Context(), claims.UserInfo))
	request.Middleware.Next()
}

func (m *Middleware) isWhitelisted(path string) bool {
	for _, pattern := range m.whitelist {
		if matches(pattern, path) {
			return true
		}
	}
	return false
}

func (m *Middleware) unauthorized(request *ghttp.Request) {
	request.Response.Status = http.StatusUnauthorized
	request.Response.WriteJson(httpresponse.Failure(http.StatusUnauthorized, "unauthorized"))
}

func bearerToken(value string) (string, bool) {
	parts := strings.Fields(value)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") && parts[1] != "" {
		return parts[1], true
	}
	return "", false
}
