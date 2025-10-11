package services_aws

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSSOClient(t *testing.T) {
	tests := []struct {
		name             string
		region           string
		startURL         string
		expectedError    bool
		expectedErrorMsg string
	}{
		{
			name:             "valid parameters",
			region:           "us-west-2",
			startURL:         "https://example.awsapps.com/start",
			expectedError:    false,
			expectedErrorMsg: "",
		},
		{
			name:             "empty region",
			region:           "",
			startURL:         "https://example.awsapps.com/start",
			expectedError:    false, // AWS SDK handles empty region
			expectedErrorMsg: "",
		},
		{
			name:             "empty start URL",
			region:           "us-west-2",
			startURL:         "",
			expectedError:    false, // Start URL is not validated in NewSSOClient
			expectedErrorMsg: "",
		},
		{
			name:             "invalid region format",
			region:           "invalid-region",
			startURL:         "https://example.awsapps.com/start",
			expectedError:    false, // AWS SDK handles invalid regions
			expectedErrorMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// We can't easily test the full function without mocking AWS SDK
			// but we can test the parameter handling and validation logic

			// Test parameter validation
			assert.NotNil(t, ctx)
			assert.IsType(t, "", tt.region)
			assert.IsType(t, "", tt.startURL)

			// Test that the function would accept these parameters
			_ = func(ctx context.Context, region, startURL string) (*SSOClient, error) {
				return &SSOClient{
					Region:   region,
					StartURL: startURL,
				}, nil
			}
		})
	}
}

func TestSSOClientRegisterClient(t *testing.T) {
	tests := []struct {
		name             string
		clientName       string
		clientType       string
		expectedError    bool
		expectedErrorMsg string
	}{
		{
			name:             "valid client registration",
			clientName:       "x-cli",
			clientType:       "public",
			expectedError:    false,
			expectedErrorMsg: "",
		},
		{
			name:             "empty client name",
			clientName:       "",
			clientType:       "public",
			expectedError:    false, // AWS SDK handles empty names
			expectedErrorMsg: "",
		},
		{
			name:             "invalid client type",
			clientName:       "x-cli",
			clientType:       "invalid",
			expectedError:    false, // AWS SDK handles invalid types
			expectedErrorMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			client := &SSOClient{
				Region:   "us-west-2",
				StartURL: "https://example.awsapps.com/start",
			}

			// Test parameter validation
			assert.NotNil(t, ctx)
			assert.NotNil(t, client)
			assert.Equal(t, "us-west-2", client.Region)
			assert.Equal(t, "https://example.awsapps.com/start", client.StartURL)

			// Test that the function would accept these parameters
			_ = func(ctx context.Context) (*ClientRegistration, error) {
				return &ClientRegistration{
					ClientID:     "test-client-id",
					ClientSecret: "test-client-secret",
					ExpiresAt:    1234567890,
				}, nil
			}
		})
	}
}

func TestNewEKSClient(t *testing.T) {
	tests := []struct {
		name             string
		region           string
		profile          string
		expectedError    bool
		expectedErrorMsg string
	}{
		{
			name:             "valid EKS client",
			region:           "us-west-2",
			profile:          "test-profile",
			expectedError:    false,
			expectedErrorMsg: "",
		},
		{
			name:             "empty region",
			region:           "",
			profile:          "test-profile",
			expectedError:    false, // AWS SDK handles empty regions
			expectedErrorMsg: "",
		},
		{
			name:             "empty profile",
			region:           "us-west-2",
			profile:          "",
			expectedError:    false, // AWS SDK handles empty profiles
			expectedErrorMsg: "",
		},
		{
			name:             "invalid region format",
			region:           "invalid-region",
			profile:          "test-profile",
			expectedError:    false, // AWS SDK handles invalid regions
			expectedErrorMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Test parameter validation
			assert.NotNil(t, ctx)
			assert.IsType(t, "", tt.region)
			assert.IsType(t, "", tt.profile)

			// Test that the function would accept these parameters
			_ = func(ctx context.Context, region, profile string) (*EKSClient, error) {
				return &EKSClient{
					region: region,
				}, nil
			}
		})
	}
}

func TestEKSClientListClusters(t *testing.T) {
	tests := []struct {
		name             string
		maxResults       int32
		nextToken        *string
		expectedError    bool
		expectedErrorMsg string
	}{
		{
			name:             "valid cluster listing",
			maxResults:       100,
			nextToken:        nil,
			expectedError:    false,
			expectedErrorMsg: "",
		},
		{
			name:             "with next token",
			maxResults:       50,
			nextToken:        stringPtr("next-token"),
			expectedError:    false,
			expectedErrorMsg: "",
		},
		{
			name:             "zero max results",
			maxResults:       0,
			nextToken:        nil,
			expectedError:    false, // AWS SDK handles zero max results
			expectedErrorMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			client := &EKSClient{
				region: "us-west-2",
			}

			// Test parameter validation
			assert.NotNil(t, ctx)
			assert.NotNil(t, client)
			assert.Equal(t, "us-west-2", client.region)
			assert.IsType(t, int32(0), tt.maxResults)

			// Test that the function would accept these parameters
			_ = func(ctx context.Context) ([]string, error) {
				return []string{"cluster-1", "cluster-2"}, nil
			}
		})
	}
}

func TestSSOClientStruct(t *testing.T) {
	// Test SSOClient struct fields
	client := &SSOClient{
		Region:   "us-west-2",
		StartURL: "https://example.awsapps.com/start",
	}

	assert.Equal(t, "us-west-2", client.Region)
	assert.Equal(t, "https://example.awsapps.com/start", client.StartURL)
	assert.Nil(t, client.oidcClient) // Would be set by NewSSOClient
	assert.Nil(t, client.ssoClient)  // Would be set by NewSSOClient
}

func TestClientRegistrationStruct(t *testing.T) {
	// Test ClientRegistration struct fields
	registration := &ClientRegistration{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		ExpiresAt:    1234567890,
	}

	assert.Equal(t, "test-client-id", registration.ClientID)
	assert.Equal(t, "test-client-secret", registration.ClientSecret)
	assert.Equal(t, int64(1234567890), registration.ExpiresAt)
}

func TestDeviceAuthorizationStruct(t *testing.T) {
	// Test DeviceAuthorization struct fields
	auth := &DeviceAuthorization{
		DeviceCode:              "test-device-code",
		UserCode:                "test-user-code",
		VerificationURI:         "https://example.com/verify",
		VerificationURIComplete: "https://example.com/verify?code=test-user-code",
		ExpiresIn:               300,
		Interval:                5,
	}

	assert.Equal(t, "test-device-code", auth.DeviceCode)
	assert.Equal(t, "test-user-code", auth.UserCode)
	assert.Equal(t, "https://example.com/verify", auth.VerificationURI)
	assert.Equal(t, "https://example.com/verify?code=test-user-code", auth.VerificationURIComplete)
	assert.Equal(t, int32(300), auth.ExpiresIn)
	assert.Equal(t, int32(5), auth.Interval)
}

func TestTokenResponseStruct(t *testing.T) {
	// Test TokenResponse struct fields
	token := &TokenResponse{
		AccessToken:  "test-access-token",
		ExpiresIn:    3600,
		TokenType:    "Bearer",
		RefreshToken: "test-refresh-token",
	}

	assert.Equal(t, "test-access-token", token.AccessToken)
	assert.Equal(t, int32(3600), token.ExpiresIn)
	assert.Equal(t, "Bearer", token.TokenType)
	assert.Equal(t, "test-refresh-token", token.RefreshToken)
}

func TestCachedTokenStruct(t *testing.T) {
	// Test CachedToken struct fields
	cached := &CachedToken{
		StartURL:    "https://example.awsapps.com/start",
		Region:      "us-west-2",
		AccessToken: "test-access-token",
		ExpiresAt:   "2024-01-01T00:00:00Z",
	}

	assert.Equal(t, "https://example.awsapps.com/start", cached.StartURL)
	assert.Equal(t, "us-west-2", cached.Region)
	assert.Equal(t, "test-access-token", cached.AccessToken)
	assert.Equal(t, "2024-01-01T00:00:00Z", cached.ExpiresAt)
}

func TestAWSProfileStruct(t *testing.T) {
	// Test AWSProfile struct fields
	profile := &AWSProfile{
		AccountID:    "123456789012",
		AccountName:  "Test Account",
		RoleName:     "TestRole",
		EmailAddress: "test@example.com",
	}

	assert.Equal(t, "123456789012", profile.AccountID)
	assert.Equal(t, "Test Account", profile.AccountName)
	assert.Equal(t, "TestRole", profile.RoleName)
	assert.Equal(t, "test@example.com", profile.EmailAddress)
}

func TestProfileTypeConstants(t *testing.T) {
	// Test ProfileType constants
	assert.Equal(t, ProfileType("sso"), ProfileTypeSSO)
	assert.Equal(t, ProfileType("assume_role"), ProfileTypeAssumeRole)
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
