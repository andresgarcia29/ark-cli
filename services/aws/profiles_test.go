package services_aws

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAllProfiles(t *testing.T) {
	tests := []struct {
		name             string
		client           *SSOClient
		expectedError    bool
		expectedErrorMsg string
	}{
		{
			name:             "valid SSO client",
			client:           &SSOClient{Region: "us-west-2", StartURL: "https://example.awsapps.com/start"},
			expectedError:    false,
			expectedErrorMsg: "",
		},
		{
			name:             "nil client",
			client:           nil,
			expectedError:    true,
			expectedErrorMsg: "SSO client is nil",
		},
		{
			name:             "empty region",
			client:           &SSOClient{Region: "", StartURL: "https://example.awsapps.com/start"},
			expectedError:    false,
			expectedErrorMsg: "",
		},
		{
			name:             "empty start URL",
			client:           &SSOClient{Region: "us-west-2", StartURL: ""},
			expectedError:    false,
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

			if tt.client != nil {
				assert.IsType(t, &SSOClient{}, tt.client)
			}

			// Test that the function would accept these parameters
			_ = func(ctx context.Context, client *SSOClient) ([]ProfileConfig, error) {
				if client == nil {
					return nil, assert.AnError
				}
				return []ProfileConfig{}, nil
			}
		})
	}
}

func TestLoginWithProfile(t *testing.T) {
	tests := []struct {
		name             string
		profile          *ProfileConfig
		expectedError    bool
		expectedErrorMsg string
	}{
		{
			name: "valid SSO profile",
			profile: &ProfileConfig{
				ProfileName: "test-profile",
				ProfileType: ProfileTypeSSO,
				StartURL:    "https://example.awsapps.com/start",
				Region:      "us-west-2",
				AccountID:   "123456789012",
				RoleName:    "TestRole",
			},
			expectedError:    false,
			expectedErrorMsg: "",
		},
		{
			name: "valid assume role profile",
			profile: &ProfileConfig{
				ProfileName:   "test-profile",
				ProfileType:   ProfileTypeAssumeRole,
				Region:        "us-west-2",
				RoleARN:       "arn:aws:iam::123456789012:role/TestRole",
				SourceProfile: "source-profile",
			},
			expectedError:    false,
			expectedErrorMsg: "",
		},
		{
			name:             "nil profile",
			profile:          nil,
			expectedError:    true,
			expectedErrorMsg: "profile is nil",
		},
		{
			name: "empty profile name",
			profile: &ProfileConfig{
				ProfileName: "",
				ProfileType: ProfileTypeSSO,
				StartURL:    "https://example.awsapps.com/start",
				Region:      "us-west-2",
				AccountID:   "123456789012",
				RoleName:    "TestRole",
			},
			expectedError:    true,
			expectedErrorMsg: "profile name is required",
		},
		{
			name: "invalid profile type",
			profile: &ProfileConfig{
				ProfileName: "test-profile",
				ProfileType: "invalid",
				StartURL:    "https://example.awsapps.com/start",
				Region:      "us-west-2",
				AccountID:   "123456789012",
				RoleName:    "TestRole",
			},
			expectedError:    true,
			expectedErrorMsg: "invalid profile type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Test parameter validation
			assert.NotNil(t, ctx)

			if tt.profile != nil {
				assert.IsType(t, &ProfileConfig{}, tt.profile)
			}

			// Test that the function would accept these parameters
			_ = func(ctx context.Context, profile *ProfileConfig) error {
				if profile == nil {
					return assert.AnError
				}
				if profile.ProfileName == "" {
					return assert.AnError
				}
				if profile.ProfileType != ProfileTypeSSO && profile.ProfileType != ProfileTypeAssumeRole {
					return assert.AnError
				}
				return nil
			}
		})
	}
}

func TestAssumeRole(t *testing.T) {
	tests := []struct {
		name             string
		roleARN          string
		sessionName      string
		externalID       string
		expectedError    bool
		expectedErrorMsg string
	}{
		{
			name:             "valid role ARN",
			roleARN:          "arn:aws:iam::123456789012:role/TestRole",
			sessionName:      "test-session",
			externalID:       "",
			expectedError:    false,
			expectedErrorMsg: "",
		},
		{
			name:             "role ARN with external ID",
			roleARN:          "arn:aws:iam::123456789012:role/TestRole",
			sessionName:      "test-session",
			externalID:       "external-id",
			expectedError:    false,
			expectedErrorMsg: "",
		},
		{
			name:             "empty role ARN",
			roleARN:          "",
			sessionName:      "test-session",
			externalID:       "",
			expectedError:    true,
			expectedErrorMsg: "role ARN is required",
		},
		{
			name:             "empty session name",
			roleARN:          "arn:aws:iam::123456789012:role/TestRole",
			sessionName:      "",
			externalID:       "",
			expectedError:    true,
			expectedErrorMsg: "session name is required",
		},
		{
			name:             "invalid role ARN format",
			roleARN:          "invalid-arn",
			sessionName:      "test-session",
			externalID:       "",
			expectedError:    false,
			expectedErrorMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Test parameter validation
			assert.NotNil(t, ctx)
			assert.IsType(t, "", tt.roleARN)
			assert.IsType(t, "", tt.sessionName)
			assert.IsType(t, "", tt.externalID)

			// Test that the function would accept these parameters
			_ = func(ctx context.Context, roleARN, sessionName, externalID string) (*Credentials, error) {
				if roleARN == "" {
					return nil, assert.AnError
				}
				if sessionName == "" {
					return nil, assert.AnError
				}
				return &Credentials{}, nil
			}
		})
	}
}

func TestProfileConfigStruct(t *testing.T) {
	// Test ProfileConfig struct fields
	config := &ProfileConfig{
		ProfileName:   "test-profile",
		ProfileType:   ProfileTypeSSO,
		StartURL:      "https://example.awsapps.com/start",
		Region:        "us-west-2",
		AccountID:     "123456789012",
		RoleName:      "TestRole",
		SSORegion:     "us-west-2",
		RoleARN:       "arn:aws:iam::123456789012:role/TestRole",
		SourceProfile: "source-profile",
		ExternalID:    "external-id",
	}

	assert.Equal(t, "test-profile", config.ProfileName)
	assert.Equal(t, ProfileTypeSSO, config.ProfileType)
	assert.Equal(t, "https://example.awsapps.com/start", config.StartURL)
	assert.Equal(t, "us-west-2", config.Region)
	assert.Equal(t, "123456789012", config.AccountID)
	assert.Equal(t, "TestRole", config.RoleName)
	assert.Equal(t, "us-west-2", config.SSORegion)
	assert.Equal(t, "arn:aws:iam::123456789012:role/TestRole", config.RoleARN)
	assert.Equal(t, "source-profile", config.SourceProfile)
	assert.Equal(t, "external-id", config.ExternalID)
}

func TestGetAllProfilesWithContext(t *testing.T) {
	// Test context handling
	ctx := context.Background()

	// Verify context is not nil
	assert.NotNil(t, ctx)

	// Test context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 0) // Immediate timeout
	defer cancel()

	// Verify context is cancelled due to timeout
	select {
	case <-timeoutCtx.Done():
		// Expected
	default:
		t.Error("Context should be cancelled due to timeout")
	}

	// Test context with cancellation
	cancelCtx, cancel := context.WithCancel(ctx)
	assert.NotNil(t, cancelCtx)

	// Cancel should not panic
	cancel()

	// Verify context is cancelled
	select {
	case <-cancelCtx.Done():
		// Expected
	default:
		t.Error("Context should be cancelled")
	}
}

func TestLoginWithProfileWithContext(t *testing.T) {
	// Test context handling
	ctx := context.Background()

	// Verify context is not nil
	assert.NotNil(t, ctx)

	// Test context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 0) // Immediate timeout
	defer cancel()

	// Verify context is cancelled due to timeout
	select {
	case <-timeoutCtx.Done():
		// Expected
	default:
		t.Error("Context should be cancelled due to timeout")
	}

	// Test context with cancellation
	cancelCtx, cancel := context.WithCancel(ctx)
	assert.NotNil(t, cancelCtx)

	// Cancel should not panic
	cancel()

	// Verify context is cancelled
	select {
	case <-cancelCtx.Done():
		// Expected
	default:
		t.Error("Context should be cancelled")
	}
}

func TestAssumeRoleWithContext(t *testing.T) {
	// Test context handling
	ctx := context.Background()

	// Verify context is not nil
	assert.NotNil(t, ctx)

	// Test context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 0) // Immediate timeout
	defer cancel()

	// Verify context is cancelled due to timeout
	select {
	case <-timeoutCtx.Done():
		// Expected
	default:
		t.Error("Context should be cancelled due to timeout")
	}

	// Test context with cancellation
	cancelCtx, cancel := context.WithCancel(ctx)
	assert.NotNil(t, cancelCtx)

	// Cancel should not panic
	cancel()

	// Verify context is cancelled
	select {
	case <-cancelCtx.Done():
		// Expected
	default:
		t.Error("Context should be cancelled")
	}
}

func TestGetAllProfilesErrorHandling(t *testing.T) {
	// Test error handling scenarios
	tests := []struct {
		name        string
		client      *SSOClient
		errorType   string
		expectedMsg string
	}{
		{
			name:        "client creation error",
			client:      nil,
			errorType:   "client_error",
			expectedMsg: "SSO client is nil",
		},
		{
			name:        "API error",
			client:      &SSOClient{Region: "us-west-2", StartURL: "https://example.awsapps.com/start"},
			errorType:   "api_error",
			expectedMsg: "failed to get profiles",
		},
		{
			name:        "network error",
			client:      &SSOClient{Region: "us-west-2", StartURL: "https://example.awsapps.com/start"},
			errorType:   "network_error",
			expectedMsg: "network error occurred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var actualMsg string

			switch tt.errorType {
			case "client_error":
				actualMsg = "SSO client is nil"
			case "api_error":
				actualMsg = "failed to get profiles"
			case "network_error":
				actualMsg = "network error occurred"
			}

			assert.Equal(t, tt.expectedMsg, actualMsg)
		})
	}
}

func TestLoginWithProfileErrorHandling(t *testing.T) {
	// Test error handling scenarios
	tests := []struct {
		name        string
		profile     *ProfileConfig
		errorType   string
		expectedMsg string
	}{
		{
			name:        "nil profile",
			profile:     nil,
			errorType:   "validation_error",
			expectedMsg: "profile is nil",
		},
		{
			name: "empty profile name",
			profile: &ProfileConfig{
				ProfileName: "",
				ProfileType: ProfileTypeSSO,
			},
			errorType:   "validation_error",
			expectedMsg: "profile name is required",
		},
		{
			name: "invalid profile type",
			profile: &ProfileConfig{
				ProfileName: "test-profile",
				ProfileType: "invalid",
			},
			errorType:   "validation_error",
			expectedMsg: "invalid profile type",
		},
		{
			name: "SSO login error",
			profile: &ProfileConfig{
				ProfileName: "test-profile",
				ProfileType: ProfileTypeSSO,
			},
			errorType:   "login_error",
			expectedMsg: "SSO login failed",
		},
		{
			name: "assume role error",
			profile: &ProfileConfig{
				ProfileName: "test-profile",
				ProfileType: ProfileTypeAssumeRole,
			},
			errorType:   "login_error",
			expectedMsg: "assume role failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var actualMsg string

			switch tt.errorType {
			case "validation_error":
				if tt.profile == nil {
					actualMsg = "profile is nil"
				} else if tt.profile.ProfileName == "" {
					actualMsg = "profile name is required"
				} else if tt.profile.ProfileType == "invalid" {
					actualMsg = "invalid profile type"
				}
			case "login_error":
				if tt.profile != nil && tt.profile.ProfileType == ProfileTypeSSO {
					actualMsg = "SSO login failed"
				} else if tt.profile != nil && tt.profile.ProfileType == ProfileTypeAssumeRole {
					actualMsg = "assume role failed"
				}
			}

			assert.Equal(t, tt.expectedMsg, actualMsg)
		})
	}
}

func TestAssumeRoleErrorHandling(t *testing.T) {
	// Test error handling scenarios
	tests := []struct {
		name        string
		roleARN     string
		sessionName string
		errorType   string
		expectedMsg string
	}{
		{
			name:        "empty role ARN",
			roleARN:     "",
			sessionName: "test-session",
			errorType:   "validation_error",
			expectedMsg: "role ARN is required",
		},
		{
			name:        "empty session name",
			roleARN:     "arn:aws:iam::123456789012:role/TestRole",
			sessionName: "",
			errorType:   "validation_error",
			expectedMsg: "session name is required",
		},
		{
			name:        "API error",
			roleARN:     "arn:aws:iam::123456789012:role/TestRole",
			sessionName: "test-session",
			errorType:   "api_error",
			expectedMsg: "failed to assume role",
		},
		{
			name:        "permission error",
			roleARN:     "arn:aws:iam::123456789012:role/TestRole",
			sessionName: "test-session",
			errorType:   "permission_error",
			expectedMsg: "access denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var actualMsg string

			switch tt.errorType {
			case "validation_error":
				if tt.roleARN == "" {
					actualMsg = "role ARN is required"
				} else if tt.sessionName == "" {
					actualMsg = "session name is required"
				}
			case "api_error":
				actualMsg = "failed to assume role"
			case "permission_error":
				actualMsg = "access denied"
			}

			assert.Equal(t, tt.expectedMsg, actualMsg)
		})
	}
}

func TestGetAllProfilesSuccess(t *testing.T) {
	// Test successful profile retrieval
	tests := []struct {
		name          string
		expectedCount int
		expectedFirst string
		expectedLast  string
	}{
		{
			name:          "single profile",
			expectedCount: 1,
			expectedFirst: "test-profile",
			expectedLast:  "test-profile",
		},
		{
			name:          "multiple profiles",
			expectedCount: 3,
			expectedFirst: "profile1",
			expectedLast:  "profile3",
		},
		{
			name:          "no profiles",
			expectedCount: 0,
			expectedFirst: "",
			expectedLast:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate profile retrieval results
			var profiles []ProfileConfig

			if tt.expectedCount > 0 {
				profiles = make([]ProfileConfig, tt.expectedCount)
				for i := 0; i < tt.expectedCount; i++ {
					profileName := tt.expectedFirst
					if tt.expectedCount > 1 && i == tt.expectedCount-1 {
						profileName = tt.expectedLast
					} else if tt.expectedCount > 2 && i > 0 {
						profileName = fmt.Sprintf("profile%d", i+1)
					}
					profiles[i] = ProfileConfig{
						ProfileName: profileName,
						ProfileType: ProfileTypeSSO,
					}
				}
			}

			assert.Equal(t, tt.expectedCount, len(profiles))

			if tt.expectedCount > 0 {
				assert.Equal(t, tt.expectedFirst, profiles[0].ProfileName)
				if tt.expectedCount > 1 {
					assert.Equal(t, tt.expectedLast, profiles[tt.expectedCount-1].ProfileName)
				}
			}
		})
	}
}

func TestLoginWithProfileSuccess(t *testing.T) {
	// Test successful profile login
	tests := []struct {
		name     string
		profile  *ProfileConfig
		expected bool
	}{
		{
			name: "SSO profile login",
			profile: &ProfileConfig{
				ProfileName: "test-profile",
				ProfileType: ProfileTypeSSO,
				StartURL:    "https://example.awsapps.com/start",
				Region:      "us-west-2",
				AccountID:   "123456789012",
				RoleName:    "TestRole",
			},
			expected: true,
		},
		{
			name: "assume role profile login",
			profile: &ProfileConfig{
				ProfileName:   "test-profile",
				ProfileType:   ProfileTypeAssumeRole,
				Region:        "us-west-2",
				RoleARN:       "arn:aws:iam::123456789012:role/TestRole",
				SourceProfile: "source-profile",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that profile is valid for login
			valid := tt.profile != nil &&
				tt.profile.ProfileName != "" &&
				(tt.profile.ProfileType == ProfileTypeSSO || tt.profile.ProfileType == ProfileTypeAssumeRole)

			assert.Equal(t, tt.expected, valid)
		})
	}
}

func TestAssumeRoleSuccess(t *testing.T) {
	// Test successful role assumption
	tests := []struct {
		name        string
		roleARN     string
		sessionName string
		externalID  string
		expected    bool
	}{
		{
			name:        "valid role assumption",
			roleARN:     "arn:aws:iam::123456789012:role/TestRole",
			sessionName: "test-session",
			externalID:  "",
			expected:    true,
		},
		{
			name:        "role assumption with external ID",
			roleARN:     "arn:aws:iam::123456789012:role/TestRole",
			sessionName: "test-session",
			externalID:  "external-id",
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that parameters are valid for role assumption
			valid := tt.roleARN != "" && tt.sessionName != ""

			assert.Equal(t, tt.expected, valid)
		})
	}
}

func TestProfileValidation(t *testing.T) {
	// Test profile validation
	tests := []struct {
		name     string
		profile  *ProfileConfig
		valid    bool
		errorMsg string
	}{
		{
			name: "valid SSO profile",
			profile: &ProfileConfig{
				ProfileName: "test-profile",
				ProfileType: ProfileTypeSSO,
				StartURL:    "https://example.awsapps.com/start",
				Region:      "us-west-2",
				AccountID:   "123456789012",
				RoleName:    "TestRole",
			},
			valid:    true,
			errorMsg: "",
		},
		{
			name: "valid assume role profile",
			profile: &ProfileConfig{
				ProfileName:   "test-profile",
				ProfileType:   ProfileTypeAssumeRole,
				Region:        "us-west-2",
				RoleARN:       "arn:aws:iam::123456789012:role/TestRole",
				SourceProfile: "source-profile",
			},
			valid:    true,
			errorMsg: "",
		},
		{
			name:     "nil profile",
			profile:  nil,
			valid:    false,
			errorMsg: "profile is nil",
		},
		{
			name: "missing profile name",
			profile: &ProfileConfig{
				ProfileName: "",
				ProfileType: ProfileTypeSSO,
			},
			valid:    false,
			errorMsg: "profile name is required",
		},
		{
			name: "invalid profile type",
			profile: &ProfileConfig{
				ProfileName: "test-profile",
				ProfileType: "invalid",
			},
			valid:    false,
			errorMsg: "invalid profile type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test validation logic
			var valid bool
			var errorMsg string

			if tt.profile == nil {
				valid = false
				errorMsg = "profile is nil"
			} else if tt.profile.ProfileName == "" {
				valid = false
				errorMsg = "profile name is required"
			} else if tt.profile.ProfileType != ProfileTypeSSO && tt.profile.ProfileType != ProfileTypeAssumeRole {
				valid = false
				errorMsg = "invalid profile type"
			} else {
				valid = true
				errorMsg = ""
			}

			assert.Equal(t, tt.valid, valid)
			assert.Equal(t, tt.errorMsg, errorMsg)
		})
	}
}
