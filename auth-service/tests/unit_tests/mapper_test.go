package unit_tests

import (
	"testing"

	"go-task-wallet-service/auth-service/internal/domain"
	"go-task-wallet-service/auth-service/internal/mapping"
	"go-task-wallet-service/auth-service/internal/pkg"
	pb "go-task-wallet-service/shared/proto/impl/auth"
)

func TestToRegisterUserDomain_Nil(t *testing.T) {
	if got := mapping.ToRegisterUserDomain(nil); got != nil {
		t.Fatalf("expected nil, got: %+v", got)
	}
}

func TestToRegisterUserDomain(t *testing.T) {
	req := &pb.RegisterUserRequest{Id: "user-1", Name: "Jane Doe", Username: "janedoe", Email: "jane@example.com", Password: "hunter22"}
	got := mapping.ToRegisterUserDomain(req)
	if got.ID != "user-1" || got.Name != "Jane Doe" || got.Username != "janedoe" || got.Email != "jane@example.com" || got.Password != "hunter22" {
		t.Fatalf("unexpected domain: %+v", got)
	}
}

func TestToRegisterUserProto_Nil(t *testing.T) {
	if got := mapping.ToRegisterUserProto(nil); got != nil {
		t.Fatalf("expected nil, got: %+v", got)
	}
}

func TestToRegisterUserProto(t *testing.T) {
	resp := &pkg.AuthenticateUserResponse{UserId: "user-1", AccessToken: "token-abc"}
	got := mapping.ToRegisterUserProto(resp)
	if got.GetId() != "user-1" || got.GetAccessToken() != "token-abc" {
		t.Fatalf("unexpected proto: %+v", got)
	}
}

func TestToLoginUserDomain_Nil(t *testing.T) {
	if got := mapping.ToLoginUserDomain(nil); got != nil {
		t.Fatalf("expected nil, got: %+v", got)
	}
}

func TestToLoginUserDomain(t *testing.T) {
	req := &pb.LoginUserRequest{Username: "janedoe", Password: "hunter22"}
	got := mapping.ToLoginUserDomain(req)
	if got.Username != "janedoe" || got.Password != "hunter22" {
		t.Fatalf("unexpected domain: %+v", got)
	}
}

func TestToLoginUserProto_Nil(t *testing.T) {
	if got := mapping.ToLoginUserProto(nil); got != nil {
		t.Fatalf("expected nil, got: %+v", got)
	}
}

func TestToLoginUserProto(t *testing.T) {
	resp := &pkg.AuthenticateUserResponse{UserId: "user-1", AccessToken: "token-abc"}
	got := mapping.ToLoginUserProto(resp)
	if got.GetId() != "user-1" || got.GetAccessToken() != "token-abc" {
		t.Fatalf("unexpected proto: %+v", got)
	}
}

func TestToRefreshTokenDomain_Nil(t *testing.T) {
	if got := mapping.ToRefreshTokenDomain(nil); got != nil {
		t.Fatalf("expected nil, got: %+v", got)
	}
}

func TestToRefreshTokenDomain(t *testing.T) {
	req := &pb.RefreshTokenRequest{RefreshToken: "refresh-abc", UserId: "user-1"}
	got := mapping.ToRefreshTokenDomain(req)
	if got.AccessToken != "refresh-abc" || got.UserId != "user-1" {
		t.Fatalf("unexpected domain: %+v", got)
	}
}

func TestToRefreshTokenProto_Nil(t *testing.T) {
	if got := mapping.ToRefreshTokenProto(nil); got != nil {
		t.Fatalf("expected nil, got: %+v", got)
	}
}

func TestToRefreshTokenProto(t *testing.T) {
	auth := &domain.UserAuth{AccessToken: "refresh-abc", UserId: "user-1"}
	got := mapping.ToRefreshTokenProto(auth)
	if got.GetRefreshToken() != "refresh-abc" || got.GetUserId() != "user-1" {
		t.Fatalf("unexpected proto: %+v", got)
	}
}
