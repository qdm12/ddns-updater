package privateip

import (
	"context"
	"errors"
	"net"
	"testing"
)

// MockInterfaceRetriever is a mock implementation of InterfaceRetriever for testing.
type MockInterfaceRetriever struct {
	InterfacesFunc func() ([]net.Interface, error)
	AddrsFunc      func(net.Interface) ([]net.Addr, error)
}

// Interfaces mocks the retrieval of network interfaces.
func (m *MockInterfaceRetriever) Interfaces() ([]net.Interface, error) {
	return m.InterfacesFunc()
}

// Addrs mocks the retrieval of addresses for a given network interface.
func (m *MockInterfaceRetriever) Addrs(iface net.Interface) ([]net.Addr, error) {
	return m.AddrsFunc(iface)
}

// TestFetcher_Success simulates the scenario where a private IP address is found.
func TestFetcher_Success(t *testing.T) {
	t.Parallel()

	privateIP := net.ParseIP("192.168.1.10").To4()
	if privateIP == nil {
		t.Fatalf("Failed to parse private IP")
	}

	mockRetriever := &MockInterfaceRetriever{
		InterfacesFunc: func() ([]net.Interface, error) {
			return []net.Interface{
				{
					Flags: net.FlagUp,
					Name:  "eth0",
				},
			}, nil
		},
		AddrsFunc: func(_ net.Interface) ([]net.Addr, error) {
			return []net.Addr{
				&net.IPNet{
					IP:   privateIP,
					Mask: net.CIDRMask(24, 32),
				},
			}, nil
		},
	}

	settings := Settings{Enabled: true}
	fetcher, err := New(settings, mockRetriever)
	if err != nil {
		t.Fatalf("Failed to create Fetcher: %v", err)
	}

	ip, err := fetcher.IP(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if ip.String() != "192.168.1.10" {
		t.Errorf("expected IP 192.168.1.10, got: %v", ip)
	}
}

// TestFetcher_NoPrivateIP simulates the scenario where no private IP address is found.
func TestFetcher_NoPrivateIP(t *testing.T) {
	t.Parallel()

	publicIP := net.ParseIP("8.8.8.8").To4()
	if publicIP == nil {
		t.Fatalf("Failed to parse public IP")
	}

	mockRetriever := &MockInterfaceRetriever{
		InterfacesFunc: func() ([]net.Interface, error) {
			return []net.Interface{
				{
					Flags: net.FlagUp,
					Name:  "eth0",
				},
			}, nil
		},
		AddrsFunc: func(_ net.Interface) ([]net.Addr, error) {
			return []net.Addr{
				&net.IPNet{
					IP:   publicIP,
					Mask: net.CIDRMask(24, 32),
				},
			}, nil
		},
	}

	settings := Settings{Enabled: true}
	fetcher, err := New(settings, mockRetriever)
	if err != nil {
		t.Fatalf("Failed to create Fetcher: %v", err)
	}

	_, err = fetcher.IP(context.Background())
	if err == nil {
		t.Fatalf("expected error, got none")
	}
	expectedErr := "no private IP address found"
	if err.Error() != expectedErr {
		t.Errorf("expected error '%s', got: %v", expectedErr, err)
	}
}

// TestFetcher_ErrorRetrievingInterfaces simulates an error when retrieving interfaces.
func TestFetcher_ErrorRetrievingInterfaces(t *testing.T) {
	t.Parallel()

	mockRetriever := &MockInterfaceRetriever{
		InterfacesFunc: func() ([]net.Interface, error) {
			return nil, errors.New("mock error retrieving interfaces")
		},
		AddrsFunc: func(_ net.Interface) ([]net.Addr, error) {
			return nil, nil
		},
	}

	settings := Settings{Enabled: true}
	fetcher, err := New(settings, mockRetriever)
	if err != nil {
		t.Fatalf("Failed to create Fetcher: %v", err)
	}

	_, err = fetcher.IP(context.Background())
	if err == nil {
		t.Fatalf("expected error, got none")
	}
	expectedErr := "mock error retrieving interfaces"
	if err.Error() != expectedErr {
		t.Errorf("expected error '%s', got: %v", expectedErr, err)
	}
}

// TestFetcher_ErrorRetrievingAddrs simulates an error when retrieving addresses.
func TestFetcher_ErrorRetrievingAddrs(t *testing.T) {
	t.Parallel()

	mockRetriever := &MockInterfaceRetriever{
		InterfacesFunc: func() ([]net.Interface, error) {
			return []net.Interface{
				{
					Flags: net.FlagUp,
					Name:  "eth0",
				},
			}, nil
		},
		AddrsFunc: func(_ net.Interface) ([]net.Addr, error) {
			return nil, errors.New("mock error retrieving addresses")
		},
	}

	settings := Settings{Enabled: true}
	fetcher, err := New(settings, mockRetriever)
	if err != nil {
		t.Fatalf("Failed to create Fetcher: %v", err)
	}

	_, err = fetcher.IP(context.Background())
	if err == nil {
		t.Fatalf("expected error, got none")
	}
	expectedErr := "mock error retrieving addresses"
	if err.Error() != expectedErr {
		t.Errorf("expected error '%s', got: %v", expectedErr, err)
	}
}

// TestFetcher_MultipleAddresses tests multiple addresses with at least one private IP.
func TestFetcher_MultipleAddresses(t *testing.T) {
	t.Parallel()

	privateIP := net.ParseIP("10.0.0.5").To4()
	publicIP := net.ParseIP("8.8.8.8").To4()
	if privateIP == nil || publicIP == nil {
		t.Fatalf("Failed to parse IPs")
	}

	mockRetriever := &MockInterfaceRetriever{
		InterfacesFunc: func() ([]net.Interface, error) {
			return []net.Interface{
				{
					Flags: net.FlagUp,
					Name:  "eth0",
				},
			}, nil
		},
		AddrsFunc: func(_ net.Interface) ([]net.Addr, error) {
			return []net.Addr{
				&net.IPNet{
					IP:   publicIP,
					Mask: net.CIDRMask(24, 32),
				},
				&net.IPNet{
					IP:   privateIP,
					Mask: net.CIDRMask(24, 32),
				},
			}, nil
		},
	}

	settings := Settings{Enabled: true}
	fetcher, err := New(settings, mockRetriever)
	if err != nil {
		t.Fatalf("Failed to create Fetcher: %v", err)
	}

	ip, err := fetcher.IP(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if ip.String() != "10.0.0.5" {
		t.Errorf("expected IP 10.0.0.5, got: %v", ip)
	}
}

// TestFetcher_InvalidIP tests an invalid IP address in the interface addresses.
func TestFetcher_InvalidIP(t *testing.T) {
	t.Parallel()

	invalidIP := []byte{0xFF, 0xFF, 0xFF, 0xFF} // Invalid IP

	mockRetriever := &MockInterfaceRetriever{
		InterfacesFunc: func() ([]net.Interface, error) {
			return []net.Interface{
				{
					Flags: net.FlagUp,
					Name:  "eth0",
				},
			}, nil
		},
		AddrsFunc: func(_ net.Interface) ([]net.Addr, error) {
			return []net.Addr{
				&net.IPNet{
					IP:   invalidIP,
					Mask: net.CIDRMask(24, 32),
				},
			}, nil
		},
	}

	settings := Settings{Enabled: true}
	fetcher, err := New(settings, mockRetriever)
	if err != nil {
		t.Fatalf("Failed to create Fetcher: %v", err)
	}

	ip, err := fetcher.IP(context.Background())
	if err == nil {
		t.Fatalf("expected error, got: %v", err)
	}
	expectedErr := "no private IP address found"
	if err.Error() != expectedErr {
		t.Errorf("expected error '%s', got: %v", expectedErr, err)
	}

	if ip.IsValid() {
		t.Errorf("expected an invalid IP address, got: %v", ip)
	}
}

// TestFetcher_Disabled tests the scenario where the Fetcher is disabled.
func TestFetcher_Disabled(t *testing.T) {
	t.Parallel()

	settings := Settings{Enabled: false}

	mockRetriever := &MockInterfaceRetriever{} // This won't be used since Fetcher is disabled.

	_, err := New(settings, mockRetriever)
	if err == nil {
		t.Fatalf("expected error, got none")
	}
	expectedErr := "private IP fetcher is disabled"
	if err.Error() != expectedErr {
		t.Errorf("expected error '%s', got: %v", expectedErr, err)
	}
}

// TestFetcher_NilRetriever tests the scenario where a nil retriever is provided.
func TestFetcher_NilRetriever(t *testing.T) {
	t.Parallel()

	settings := Settings{Enabled: true}

	_, err := New(settings, nil)
	if err == nil {
		t.Fatalf("expected error, got none")
	}
	expectedErr := "interface retriever cannot be nil"
	if err.Error() != expectedErr {
		t.Errorf("expected error '%s', got: %v", expectedErr, err)
	}
}
