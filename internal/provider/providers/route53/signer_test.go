package route53

import (
	"encoding/json"
	"net/netip"
	"testing"

	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_WithAccessKey(t *testing.T) {
	t.Parallel()

	settings := map[string]interface{}{
		"access_key": "AKIDEXAMPLE",
		"secret_key": "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
		"zone_id":    "Z123456789",
		"ttl":        300,
	}

	data, err := json.Marshal(settings)
	require.NoError(t, err)

	provider, err := New(data, "example.com", "test", ipversion.IP4, netip.Prefix{})
	require.NoError(t, err)
	assert.NotNil(t, provider)

	// Check basic fields
	assert.Equal(t, "example.com", provider.domain)
	assert.Equal(t, "test", provider.owner)
	assert.Equal(t, "Z123456789", provider.zoneID)
	assert.Equal(t, uint32(300), provider.ttl)
	assert.Equal(t, ipversion.IP4, provider.ipVersion)

	// Check that static credentials are stored
	assert.Equal(t, "AKIDEXAMPLE", provider.accessKey)
	assert.Equal(t, "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY", provider.secretKey)
	assert.NotNil(t, provider.session)
}

func TestNew_WithProfile(t *testing.T) {
	t.Parallel()

	settings := map[string]interface{}{
		"aws_profile": "test-profile",
		"zone_id":     "Z123456789",
	}

	data, err := json.Marshal(settings)
	require.NoError(t, err)

	// This test will fail if the profile doesn't exist, but that's expected in CI
	// We're just testing that the code path works correctly
	_, err = New(data, "example.com", "test", ipversion.IP4, netip.Prefix{})
	// Should either succeed or fail with a specific AWS profile error
	if err != nil {
		// Expected in environments without the test profile
		assert.Contains(t, err.Error(), "creating AWS session")
	}
}

func TestNew_WithValidProfile_MockScenario(t *testing.T) {
	t.Parallel()

	// Test the validation logic for profile scenarios
	// This tests the validateSettings function with profile parameters

	testCases := []struct {
		name        string
		domain      string
		awsProfile  string
		zoneID      string
		expectError bool
		errorType   string
	}{
		{
			name:        "valid profile with zone ID",
			domain:      "example.com",
			awsProfile:  "my-profile",
			zoneID:      "Z123456789",
			expectError: false,
		},
		{
			name:        "profile without zone ID should fail",
			domain:      "example.com",
			awsProfile:  "my-profile",
			zoneID:      "",
			expectError: true,
			errorType:   "zone identifier is not set",
		},
		{
			name:        "invalid domain with profile",
			domain:      "",
			awsProfile:  "my-profile",
			zoneID:      "Z123456789",
			expectError: true,
			errorType:   "domain is not valid",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateSettings(tc.domain, "", "", tc.awsProfile, tc.zoneID)
			if tc.expectError {
				assert.Error(t, err)
				if tc.errorType != "" {
					assert.Contains(t, err.Error(), tc.errorType)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNew_ProfileVsAccessKey_Priority(t *testing.T) {
	t.Parallel()

	// Test what happens when both profile and access keys are provided
	// Profile should take priority
	settings := map[string]interface{}{
		"aws_profile": "test-profile",
		"access_key":  "AKIDEXAMPLE",
		"secret_key":  "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
		"zone_id":     "Z123456789",
	}

	data, err := json.Marshal(settings)
	require.NoError(t, err)

	provider, err := New(data, "example.com", "test", ipversion.IP4, netip.Prefix{})

	// Should fail because profile doesn't exist, but importantly:
	// - It should attempt to use the profile (not the access keys)
	// - The accessKey and secretKey fields should be empty (not populated from settings)
	if err != nil {
		assert.Contains(t, err.Error(), "creating AWS session")
	} else {
		// If somehow the profile exists and succeeds
		assert.Empty(t, provider.accessKey, "When profile is provided, accessKey should be empty")
		assert.Empty(t, provider.secretKey, "When profile is provided, secretKey should be empty")
		assert.NotNil(t, provider.session, "Session should be created when using profile")
	}
}

func TestNew_DefaultTTL(t *testing.T) {
	t.Parallel()

	settings := map[string]interface{}{
		"access_key": "AKIDEXAMPLE",
		"secret_key": "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
		"zone_id":    "Z123456789",
		// No TTL specified
	}

	data, err := json.Marshal(settings)
	require.NoError(t, err)

	provider, err := New(data, "example.com", "test", ipversion.IP4, netip.Prefix{})
	require.NoError(t, err)

	// Should use default TTL of 300
	assert.Equal(t, uint32(300), provider.ttl)
}

func TestValidateSettings_AccessKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		domain      string
		accessKey   string
		secretKey   string
		awsProfile  string
		zoneID      string
		expectError bool
	}{
		{
			name:        "valid access key setup",
			domain:      "example.com",
			accessKey:   "AKIDEXAMPLE",
			secretKey:   "secret",
			zoneID:      "Z123456789",
			expectError: false,
		},
		{
			name:        "missing access key",
			domain:      "example.com",
			secretKey:   "secret",
			zoneID:      "Z123456789",
			expectError: true,
		},
		{
			name:        "missing secret key",
			domain:      "example.com",
			accessKey:   "AKIDEXAMPLE",
			zoneID:      "Z123456789",
			expectError: true,
		},
		{
			name:        "missing zone ID",
			domain:      "example.com",
			accessKey:   "AKIDEXAMPLE",
			secretKey:   "secret",
			expectError: true,
		},
		{
			name:        "valid profile setup",
			domain:      "example.com",
			awsProfile:  "test-profile",
			zoneID:      "Z123456789",
			expectError: false,
		},
		{
			name:        "profile missing zone ID",
			domain:      "example.com",
			awsProfile:  "test-profile",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSettings(tt.domain, tt.accessKey, tt.secretKey, tt.awsProfile, tt.zoneID)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProvider_Methods(t *testing.T) {
	t.Parallel()

	settings := map[string]interface{}{
		"access_key": "AKIDEXAMPLE",
		"secret_key": "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
		"zone_id":    "Z123456789",
	}

	data, err := json.Marshal(settings)
	require.NoError(t, err)

	provider, err := New(data, "example.com", "test", ipversion.IP4, netip.Prefix{})
	require.NoError(t, err)

	// Test interface methods
	assert.Equal(t, "example.com", provider.Domain())
	assert.Equal(t, "test", provider.Owner())
	assert.Equal(t, ipversion.IP4, provider.IPVersion())
	assert.False(t, provider.Proxied())
	assert.Equal(t, "test.example.com", provider.BuildDomainName())

	// Test string representation
	assert.Contains(t, provider.String(), "example.com")
	assert.Contains(t, provider.String(), "test")
	assert.Contains(t, provider.String(), "route53")

	// Test HTML representation
	html := provider.HTML()
	assert.Contains(t, html.Domain, "test.example.com")
	assert.Equal(t, "test", html.Owner)
	assert.Contains(t, html.Provider, "Amazon Route 53")
	assert.Equal(t, "ipv4", html.IPVersion)
}
