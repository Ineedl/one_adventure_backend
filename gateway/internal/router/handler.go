package router

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gogf/gf/v2/net/ghttp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"one_adventure_servicekit/httpresponse"
	servicetoken "one_adventure_servicekit/token"

	"one_adventure_gateway/internal/service"
)

type Handler struct {
	resolver service.ServiceResolver
	routes   RouteTable
}

func NewHandler(resolver service.ServiceResolver, routes RouteTable) *Handler {
	return &Handler{resolver: resolver, routes: routes}
}

func (h *Handler) Handle(request *ghttp.Request) {
	httpStatus, response := h.dispatch(
		request.Context(),
		request.Method,
		request.URL.Path,
		request.GetBody(),
	)
	request.Response.Status = httpStatus
	request.Response.WriteJson(response)
}

func (h *Handler) dispatch(ctx context.Context, method, uri string, body []byte) (int, httpresponse.Response) {
	key, err := parseRouteKey(uri)
	if err != nil {
		return http.StatusNotFound, httpresponse.Failure(http.StatusNotFound, err.Error())
	}
	connection, err := h.resolver.ResolveService(key.Service)
	if err != nil {
		if errors.Is(err, service.ErrServiceUnavailable) {
			return http.StatusServiceUnavailable, httpresponse.Failure(http.StatusServiceUnavailable, err.Error())
		}
		return http.StatusInternalServerError, httpresponse.Failure(http.StatusInternalServerError, "resolve service failed")
	}
	route, ok := h.routes[key]
	if !ok {
		return http.StatusNotFound, httpresponse.Failure(http.StatusNotFound, "route not found")
	}
	if method != route.Method {
		return http.StatusMethodNotAllowed, httpresponse.Failure(http.StatusMethodNotAllowed, "method not allowed")
	}

	rpcRequest := route.NewRequest()
	if len(body) == 0 {
		return http.StatusBadRequest, httpresponse.Failure(http.StatusBadRequest, "request body is required")
	}
	if err = json.Unmarshal(body, rpcRequest); err != nil {
		return http.StatusBadRequest, httpresponse.Failure(http.StatusBadRequest, "invalid request body")
	}
	ctx, err = servicetoken.WithOutgoingUserInfo(ctx)
	if err != nil {
		return http.StatusInternalServerError, httpresponse.Failure(http.StatusInternalServerError, "create user metadata failed")
	}
	rpcResponse, err := route.Invoke(ctx, connection, rpcRequest)
	if err != nil {
		return rpcErrorResponse(err)
	}
	return http.StatusOK, httpresponse.Success(rpcResponse)
}

func parseRouteKey(uri string) (RouteKey, error) {
	parts := strings.Split(strings.Trim(uri, "/"), "/")
	if len(parts) < 4 || !strings.EqualFold(parts[1], "api") {
		return RouteKey{}, errors.New("uri must match /{service}/api/{version}/{route}")
	}
	serviceName := strings.ToLower(strings.TrimSpace(parts[0]))
	version := strings.ToLower(strings.TrimSpace(parts[2]))
	routePath := strings.ToLower(strings.Join(parts[3:], "/"))
	if serviceName == "" || version == "" || routePath == "" {
		return RouteKey{}, errors.New("uri must match /{service}/api/{version}/{route}")
	}
	return RouteKey{Service: serviceName, Version: version, Path: routePath}, nil
}

func rpcErrorResponse(err error) (int, httpresponse.Response) {
	rpcStatus, ok := status.FromError(err)
	if !ok {
		return http.StatusInternalServerError, httpresponse.Failure(http.StatusInternalServerError, "rpc request failed")
	}
	httpStatus := http.StatusInternalServerError
	switch rpcStatus.Code() {
	case codes.InvalidArgument:
		httpStatus = http.StatusBadRequest
	case codes.Unauthenticated:
		httpStatus = http.StatusUnauthorized
	case codes.PermissionDenied:
		httpStatus = http.StatusForbidden
	case codes.NotFound:
		httpStatus = http.StatusNotFound
	case codes.ResourceExhausted:
		httpStatus = http.StatusTooManyRequests
	case codes.Unavailable:
		httpStatus = http.StatusServiceUnavailable
	case codes.DeadlineExceeded:
		httpStatus = http.StatusGatewayTimeout
	}
	return httpStatus, httpresponse.Failure(httpStatus, rpcStatus.Message())
}
