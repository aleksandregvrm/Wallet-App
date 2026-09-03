package integration_tests

import (
	"net/http"
	"testing"

	"go-task-wallet-service/api-gateway/tests/fixtures"
)

func TestHealth_Liveness(t *testing.T) {
	server := newTestServer(t, &fixtures.FakeAuthServiceServer{}, &fixtures.FakeWalletServiceServer{})

	resp, err := http.Get(server.URL + "/health/liveness")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got: %d", resp.StatusCode)
	}
}

func TestHealth_Readiness(t *testing.T) {
	server := newTestServer(t, &fixtures.FakeAuthServiceServer{}, &fixtures.FakeWalletServiceServer{})

	resp, err := http.Get(server.URL + "/health/readiness")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got: %d", resp.StatusCode)
	}
}
