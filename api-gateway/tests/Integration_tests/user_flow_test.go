package integration_tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"go-task-wallet-service/api-gateway/tests/fixtures"
	pbAuth "go-task-wallet-service/shared/proto/impl/auth"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func postJSON(t *testing.T, url string, body any, headers map[string]string) *http.Response {
	t.Helper()

	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return resp
}

func decodeJSON(t *testing.T, resp *http.Response, out any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
}

func TestUserFlow_Register_Success(t *testing.T) {
	authSrv := &fixtures.FakeAuthServiceServer{
		RegisterUserFunc: func(ctx context.Context, in *pbAuth.RegisterUserRequest) (*pbAuth.RegisterUserResponse, error) {
			if in.GetUsername() != "janedoe" || in.GetEmail() != "jane@example.com" {
				t.Fatalf("unexpected request: %+v", in)
			}
			return &pbAuth.RegisterUserResponse{Id: "user-1", AccessToken: "token-abc"}, nil
		},
	}
	server := newTestServer(t, authSrv, &fixtures.FakeWalletServiceServer{})

	resp := postJSON(t, server.URL+"/user/register", map[string]string{
		"name": "Jane Doe", "username": "janedoe", "email": "jane@example.com", "password": "hunter22",
	}, nil)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got: %d", resp.StatusCode)
	}
	var body struct {
		Status      string `json:"status"`
		AccessToken string `json:"acccess_token"`
	}
	decodeJSON(t, resp, &body)
	if body.Status != "user registered" || body.AccessToken != "token-abc" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestUserFlow_Register_BackendError(t *testing.T) {
	authSrv := &fixtures.FakeAuthServiceServer{
		RegisterUserFunc: func(ctx context.Context, in *pbAuth.RegisterUserRequest) (*pbAuth.RegisterUserResponse, error) {
			return nil, status.Error(codes.AlreadyExists, "username taken")
		},
	}
	server := newTestServer(t, authSrv, &fixtures.FakeWalletServiceServer{})

	resp := postJSON(t, server.URL+"/user/register", map[string]string{
		"name": "Jane Doe", "username": "janedoe", "email": "jane@example.com", "password": "hunter22",
	}, nil)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got: %d", resp.StatusCode)
	}
}

func TestUserFlow_Register_MalformedBody(t *testing.T) {
	server := newTestServer(t, &fixtures.FakeAuthServiceServer{}, &fixtures.FakeWalletServiceServer{})

	req, _ := http.NewRequest(http.MethodPost, server.URL+"/user/register", bytes.NewReader([]byte("{not-json")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got: %d", resp.StatusCode)
	}
}

func TestUserFlow_Login_Success(t *testing.T) {
	authSrv := &fixtures.FakeAuthServiceServer{
		LoginUserFunc: func(ctx context.Context, in *pbAuth.LoginUserRequest) (*pbAuth.LoginUserResponse, error) {
			if in.GetUsername() != "janedoe" || in.GetPassword() != "hunter22" {
				t.Fatalf("unexpected request: %+v", in)
			}
			return &pbAuth.LoginUserResponse{Id: "user-1", AccessToken: "token-abc"}, nil
		},
	}
	server := newTestServer(t, authSrv, &fixtures.FakeWalletServiceServer{})

	resp := postJSON(t, server.URL+"/user/login", map[string]string{
		"username": "janedoe", "password": "hunter22",
	}, nil)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got: %d", resp.StatusCode)
	}
	var body struct {
		Status      string `json:"status"`
		AccessToken string `json:"acccess_token"`
	}
	decodeJSON(t, resp, &body)
	if body.Status != "user authenticated" || body.AccessToken != "token-abc" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestUserFlow_Login_InvalidCredentials(t *testing.T) {
	authSrv := &fixtures.FakeAuthServiceServer{
		LoginUserFunc: func(ctx context.Context, in *pbAuth.LoginUserRequest) (*pbAuth.LoginUserResponse, error) {
			return nil, status.Error(codes.Unauthenticated, "invalid credentials")
		},
	}
	server := newTestServer(t, authSrv, &fixtures.FakeWalletServiceServer{})

	resp := postJSON(t, server.URL+"/user/login", map[string]string{
		"username": "janedoe", "password": "wrongpass",
	}, nil)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got: %d", resp.StatusCode)
	}
}

func TestUserFlow_Refresh_AlwaysUnauthorized(t *testing.T) {
	server := newTestServer(t, &fixtures.FakeAuthServiceServer{}, &fixtures.FakeWalletServiceServer{})

	resp := postJSON(t, server.URL+"/user/refresh", map[string]string{
		"refresh_token": "some-refresh-token",
	}, nil)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got: %d", resp.StatusCode)
	}
}
