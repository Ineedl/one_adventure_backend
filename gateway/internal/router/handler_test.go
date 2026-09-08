package router

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	promotionpb "one_adventure_rpc/proto/promotion"
	userpb "one_adventure_rpc/proto/user"

	"one_adventure_gateway/internal/service"
	servicetoken "one_adventure_servicekit/token"
)

type fakeResolver struct {
	serviceName string
	connection  grpc.ClientConnInterface
	err         error
}

func (r *fakeResolver) ResolveService(serviceName string) (grpc.ClientConnInterface, error) {
	r.serviceName = serviceName
	return r.connection, r.err
}

type fakeConnection struct {
	invoke func(method string, request, response any) error
}

func (c *fakeConnection) Invoke(_ context.Context, method string, request, response any, _ ...grpc.CallOption) error {
	return c.invoke(method, request, response)
}

func (c *fakeConnection) NewStream(context.Context, *grpc.StreamDesc, string, ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, errors.New("streams are not supported by fake connection")
}

func TestHandlerDispatchesLoginByURI(t *testing.T) {
	connection := &fakeConnection{invoke: func(method string, request, response any) error {
		if method != userpb.UserService_Login_FullMethodName {
			t.Fatalf("gRPC method = %q", method)
		}
		loginRequest := request.(*userpb.LoginReq)
		if loginRequest.GetUsername() != "alice" || loginRequest.GetPassword() != "secret" {
			t.Fatalf("login request = %#v", loginRequest)
		}
		loginResponse := response.(*userpb.LoginResp)
		loginResponse.Token = "access"
		loginResponse.RefreshToken = "refresh"
		return nil
	}}
	resolver := &fakeResolver{connection: connection}
	handler := NewHandler(resolver, DefaultRouteTable())

	httpStatus, response := handler.dispatch(
		context.Background(), http.MethodPost, "/user/api/v1/login",
		[]byte(`{"username":"alice","password":"secret"}`),
	)
	if httpStatus != http.StatusOK || response.Code != 0 || response.Msg != "" {
		t.Fatalf("dispatch status = %d, response = %#v", httpStatus, response)
	}
	if resolver.serviceName != "user" {
		t.Fatalf("resolved service = %q", resolver.serviceName)
	}
	loginResponse := response.Data.(*userpb.LoginResp)
	if loginResponse.GetToken() != "access" || loginResponse.GetRefreshToken() != "refresh" {
		t.Fatalf("response data = %#v", loginResponse)
	}
}

func TestHandlerDispatchesRefreshToken(t *testing.T) {
	connection := &fakeConnection{invoke: func(method string, request, response any) error {
		if method != userpb.UserService_RefreshToken_FullMethodName {
			t.Fatalf("gRPC method = %q", method)
		}
		if request.(*userpb.RefreshTokenReq).GetRefreshToken() != "refresh" {
			t.Fatalf("refresh request = %#v", request)
		}
		response.(*userpb.RefreshTokenResp).Token = "new-access"
		return nil
	}}
	handler := NewHandler(&fakeResolver{connection: connection}, DefaultRouteTable())

	httpStatus, response := handler.dispatch(
		context.Background(), http.MethodPost, "/user/api/v1/refresh-token",
		[]byte(`{"refresh_token":"refresh"}`),
	)
	if httpStatus != http.StatusOK || response.Data.(*userpb.RefreshTokenResp).GetToken() != "new-access" {
		t.Fatalf("dispatch status = %d, response = %#v", httpStatus, response)
	}
}

func TestHandlerRejectsInvalidRoutesAndMethods(t *testing.T) {
	handler := NewHandler(&fakeResolver{}, DefaultRouteTable())
	tests := []struct {
		method string
		uri    string
		want   int
	}{
		{method: http.MethodPost, uri: "/user/v1/login", want: http.StatusNotFound},
		{method: http.MethodPost, uri: "/user/api/v2/login", want: http.StatusNotFound},
		{method: http.MethodGet, uri: "/user/api/v1/login", want: http.StatusMethodNotAllowed},
	}
	for _, test := range tests {
		httpStatus, response := handler.dispatch(context.Background(), test.method, test.uri, []byte(`{}`))
		if httpStatus != test.want || response.Code != test.want {
			t.Fatalf("dispatch(%q) status = %d, response = %#v", test.uri, httpStatus, response)
		}
	}
}

func TestHandlerRequiresAdminOnlyForAdminRoutes(t *testing.T) {
	key := RouteKey{Service: "user", Version: "v1", Path: "login"}
	invoked := false
	routes := RouteTable{key: {
		Method:     http.MethodPost,
		IsAdmin:    true,
		NewRequest: func() any { return &userpb.LoginReq{} },
		Invoke: func(context.Context, grpc.ClientConnInterface, any) (any, error) {
			invoked = true
			return &userpb.LoginResp{}, nil
		},
	}}
	resolver := &fakeResolver{connection: &fakeConnection{}}
	handler := NewHandler(resolver, routes)

	for _, ctx := range []context.Context{
		context.Background(),
		servicetoken.WithUserInfo(context.Background(), servicetoken.UserInfo{ID: 1, IsAdmin: false}),
	} {
		httpStatus, response := handler.dispatch(ctx, http.MethodPost, "/user/api/v1/login", []byte(`{}`))
		if httpStatus != http.StatusForbidden || response.Code != http.StatusForbidden || response.Msg != "permission denied" {
			t.Fatalf("non-admin status = %d, response = %#v", httpStatus, response)
		}
	}
	if invoked || resolver.serviceName != "" {
		t.Fatal("admin-only route reached resolver or downstream for non-admin user")
	}

	ctx := servicetoken.WithUserInfo(context.Background(), servicetoken.UserInfo{ID: 1, IsAdmin: true})
	httpStatus, response := handler.dispatch(ctx, http.MethodPost, "/user/api/v1/login", []byte(`{}`))
	if httpStatus != http.StatusOK || response.Code != 0 || !invoked {
		t.Fatalf("admin status = %d, response = %#v, invoked = %v", httpStatus, response, invoked)
	}
}

func TestHandlerDispatchesPromotionStockRefreshForAdmin(t *testing.T) {
	connection := &fakeConnection{invoke: func(method string, request, response any) error {
		if method != promotionpb.PromotionService_PromotionStockRefresh_FullMethodName {
			t.Fatalf("gRPC method = %q", method)
		}
		if request.(*promotionpb.PromotionStockRefreshReq).GetPromotionId() != 7 {
			t.Fatalf("refresh request = %#v", request)
		}
		response.(*promotionpb.PromotionStockRefreshResp).RefreshedCount = 3
		return nil
	}}
	handler := NewHandler(&fakeResolver{connection: connection}, DefaultRouteTable())
	ctx := servicetoken.WithUserInfo(context.Background(), servicetoken.UserInfo{ID: 1, IsAdmin: true})

	httpStatus, response := handler.dispatch(
		ctx, http.MethodPost, "/promotion/api/v1/stock-refresh", []byte(`{"promotion_id":7}`),
	)
	result, ok := response.Data.(*promotionpb.PromotionStockRefreshResp)
	if httpStatus != http.StatusOK || !ok || result.GetRefreshedCount() != 3 {
		t.Fatalf("dispatch status = %d, response = %#v", httpStatus, response)
	}
}

func TestHandlerMapsUnavailableAndGRPCErrors(t *testing.T) {
	handler := NewHandler(&fakeResolver{err: service.ErrServiceUnavailable}, DefaultRouteTable())
	httpStatus, response := handler.dispatch(
		context.Background(), http.MethodPost, "/user/api/v1/login", []byte(`{}`),
	)
	if httpStatus != http.StatusServiceUnavailable || response.Code != http.StatusServiceUnavailable {
		t.Fatalf("unavailable status = %d, response = %#v", httpStatus, response)
	}

	connection := &fakeConnection{invoke: func(string, any, any) error {
		return status.Error(codes.Unauthenticated, "invalid username or password")
	}}
	handler = NewHandler(&fakeResolver{connection: connection}, DefaultRouteTable())
	httpStatus, response = handler.dispatch(
		context.Background(), http.MethodPost, "/user/api/v1/login", []byte(`{}`),
	)
	if httpStatus != http.StatusUnauthorized || response.Msg != "invalid username or password" {
		t.Fatalf("authentication status = %d, response = %#v", httpStatus, response)
	}
}

var _ grpc.ClientConnInterface = (*fakeConnection)(nil)
var _ grpc.ClientStream = (*fakeClientStream)(nil)

type fakeClientStream struct{}

func (*fakeClientStream) Header() (metadata.MD, error) { return nil, nil }
func (*fakeClientStream) Trailer() metadata.MD         { return nil }
func (*fakeClientStream) CloseSend() error             { return nil }
func (*fakeClientStream) Context() context.Context     { return context.Background() }
func (*fakeClientStream) SendMsg(any) error            { return nil }
func (*fakeClientStream) RecvMsg(any) error            { return nil }
