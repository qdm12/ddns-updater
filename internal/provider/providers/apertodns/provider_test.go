package apertodns

import (
	"context"
	"encoding/json"
	"net/http"
	"net/netip"
	"testing"
	"time"

	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

func TestUpdate(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Real test token for brave-panda.apertodns.com
	token := "test-doc-2956db1b53af3cbe601202784062ce05"

	data, _ := json.Marshal(map[string]string{
		"token": token,
	})

	p, err := New(data, "apertodns.com", "brave-panda", ipversion.IP4, netip.Prefix{})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Create HTTP client
	client := &http.Client{Timeout: 20 * time.Second}

	// Test with a real IP
	testIP := netip.MustParseAddr("93.44.241.82")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Call Update (uses modern first, fallback if needed)
	returnedIP, err := p.Update(ctx, client, testIP)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if returnedIP.Compare(testIP) != 0 {
		t.Errorf("Expected IP %s, got %s", testIP, returnedIP)
	}

	t.Logf("SUCCESS: Updated to %s", returnedIP)
}

func TestUpdateModern(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Real test token for brave-panda.apertodns.com
	token := "test-doc-2956db1b53af3cbe601202784062ce05"

	data, _ := json.Marshal(map[string]string{
		"token": token,
	})

	p, err := New(data, "apertodns.com", "brave-panda", ipversion.IP4, netip.Prefix{})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	client := &http.Client{Timeout: 20 * time.Second}
	testIP := netip.MustParseAddr("93.44.241.82")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Call updateModern directly
	returnedIP, err := p.updateModern(ctx, client, testIP)
	if err != nil {
		t.Fatalf("Modern update failed: %v", err)
	}

	if returnedIP.Compare(testIP) != 0 {
		t.Errorf("Expected IP %s, got %s", testIP, returnedIP)
	}

	t.Logf("SUCCESS: Modern updated to %s", returnedIP)
}

func TestUpdateLegacy(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Real test token for brave-panda.apertodns.com
	token := "test-doc-2956db1b53af3cbe601202784062ce05"

	data, _ := json.Marshal(map[string]string{
		"token": token,
	})

	p, err := New(data, "apertodns.com", "brave-panda", ipversion.IP4, netip.Prefix{})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	client := &http.Client{Timeout: 20 * time.Second}
	testIP := netip.MustParseAddr("93.44.241.82")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Call updateLegacy directly
	returnedIP, err := p.updateLegacy(ctx, client, testIP)
	if err != nil {
		t.Fatalf("Legacy update failed: %v", err)
	}

	if returnedIP.Compare(testIP) != 0 {
		t.Errorf("Expected IP %s, got %s", testIP, returnedIP)
	}

	t.Logf("SUCCESS: Legacy updated to %s", returnedIP)
}

func TestUpdateInvalidToken(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Invalid token
	data, _ := json.Marshal(map[string]string{
		"token": "invalid_token_12345",
	})

	p, err := New(data, "apertodns.com", "brave-panda", ipversion.IP4, netip.Prefix{})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	client := &http.Client{Timeout: 20 * time.Second}
	testIP := netip.MustParseAddr("93.44.241.82")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Should fail on modern and NOT fallback (auth error)
	_, err = p.Update(ctx, client, testIP)
	if err == nil {
		t.Fatal("Expected error for invalid token, got nil")
	}

	t.Logf("SUCCESS: Got expected error (no fallback for auth): %v", err)
}

func TestUpdateHostnameNotFound(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Valid token but wrong hostname
	token := "test-doc-2956db1b53af3cbe601202784062ce05"

	data, _ := json.Marshal(map[string]string{
		"token": token,
	})

	p, err := New(data, "apertodns.com", "nonexistent-hostname-xyz", ipversion.IP4, netip.Prefix{})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	client := &http.Client{Timeout: 20 * time.Second}
	testIP := netip.MustParseAddr("93.44.241.82")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Should fail on modern and NOT fallback (hostname not found error)
	_, err = p.Update(ctx, client, testIP)
	if err == nil {
		t.Fatal("Expected error for nonexistent hostname, got nil")
	}

	t.Logf("SUCCESS: Got expected error (no fallback for hostname): %v", err)
}
