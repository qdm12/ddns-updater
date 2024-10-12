package privateip

import (
	"errors"
	"net"
	"testing"
)

// Backup the original InterfaceAddrs to restore after tests
var originalInterfaceAddrs = InterfaceAddrs

// mockInterfaceAddrs is used to mock net.InterfaceAddrs in tests.
func mockInterfaceAddrs(addrs []net.Addr, err error) {
	InterfaceAddrs = func() ([]net.Addr, error) {
		return addrs, err
	}
}

// Restore the original InterfaceAddrs after a test
func restoreInterfaceAddrs() {
	InterfaceAddrs = originalInterfaceAddrs
}

// TestFetchPrivateIP_Success simulates the scenario where a private IP address is found.
func TestFetchPrivateIP_Success(t *testing.T) {
	defer restoreInterfaceAddrs()

	privateIP := net.ParseIP("192.168.1.10").To4()
	if privateIP == nil {
		t.Fatalf("Failed to parse private IP")
	}

	mockInterfaceAddrs([]net.Addr{
		&net.IPNet{
			IP:   privateIP,
			Mask: net.CIDRMask(24, 32),
		},
	}, nil)

	ip, err := fetch()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if ip.String() != "192.168.1.10" {
		t.Errorf("expected IP 192.168.1.10, got: %v", ip)
	}
}

// TestFetchPrivateIP_NoPrivateIP simulates the scenario where no private IP address is found.
func TestFetchPrivateIP_NoPrivateIP(t *testing.T) {
	defer restoreInterfaceAddrs()

	publicIP := net.ParseIP("8.8.8.8").To4()
	if publicIP == nil {
		t.Fatalf("Failed to parse public IP")
	}

	mockInterfaceAddrs([]net.Addr{
		&net.IPNet{
			IP:   publicIP,
			Mask: net.CIDRMask(24, 32),
		},
	}, nil)

	ip, err := fetch()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if ip.IsValid() && ip.IsPrivate() {
		t.Errorf("expected an invalid IP address, got: %v", ip)
	}
}

// TestFetchPrivateIP_Error simulates an error scenario when retrieving interface addresses.
func TestFetchPrivateIP_Error(t *testing.T) {
	defer restoreInterfaceAddrs()

	mockInterfaceAddrs(nil, errors.New("mock error"))

	_, err := fetch()
	if err == nil {
		t.Fatalf("expected error, got none")
	}
	if err.Error() != "mock error" {
		t.Errorf("expected error 'mock error', got: %v", err)
	}
}

// TestFetchPrivateIP_MultipleAddresses tests multiple addresses with at least one private IP.
func TestFetchPrivateIP_MultipleAddresses(t *testing.T) {
	defer restoreInterfaceAddrs()

	privateIP := net.ParseIP("10.0.0.5").To4()
	publicIP := net.ParseIP("8.8.8.8").To4()
	if privateIP == nil || publicIP == nil {
		t.Fatalf("Failed to parse IPs")
	}

	mockInterfaceAddrs([]net.Addr{
		&net.IPNet{
			IP:   publicIP,
			Mask: net.CIDRMask(24, 32),
		},
		&net.IPNet{
			IP:   privateIP,
			Mask: net.CIDRMask(24, 32),
		},
	}, nil)

	ip, err := fetch()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if ip.String() != "10.0.0.5" {
		t.Errorf("expected IP 10.0.0.5, got: %v", ip)
	}
}

// TestFetchPrivateIP_InvalidIP tests an invalid IP address in the interface addresses.
func TestFetchPrivateIP_InvalidIP(t *testing.T) {
	defer restoreInterfaceAddrs()

	invalidIP := []byte{0xFF, 0xFF, 0xFF, 0xFF} // Invalid IP
	mockInterfaceAddrs([]net.Addr{
		&net.IPNet{
			IP:   invalidIP,
			Mask: net.CIDRMask(24, 32),
		},
	}, nil)

	ip, err := fetch()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if ip.IsValid() {
		t.Errorf("expected an invalid IP address, got: %v", ip)
	}
}
