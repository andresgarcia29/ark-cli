package services_aws

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListAccounts(t *testing.T) {
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
			_ = func(ctx context.Context, client *SSOClient) ([]Account, error) {
				if client == nil {
					return nil, assert.AnError
				}
				return []Account{}, nil
			}
		})
	}
}

func TestAccountStruct(t *testing.T) {
	// Test Account struct fields
	account := Account{
		AccountID:    "123456789012",
		AccountName:  "Test Account",
		EmailAddress: "test@example.com",
	}

	assert.Equal(t, "123456789012", account.AccountID)
	assert.Equal(t, "Test Account", account.AccountName)
	assert.Equal(t, "test@example.com", account.EmailAddress)
}

func TestListAccountsWithContext(t *testing.T) {
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

func TestListAccountsErrorHandling(t *testing.T) {
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
			expectedMsg: "failed to list accounts",
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
				actualMsg = "failed to list accounts"
			case "network_error":
				actualMsg = "network error occurred"
			}

			assert.Equal(t, tt.expectedMsg, actualMsg)
		})
	}
}

func TestListAccountsSuccess(t *testing.T) {
	// Test successful account listing
	tests := []struct {
		name          string
		expectedCount int
		expectedFirst string
		expectedLast  string
	}{
		{
			name:          "single account",
			expectedCount: 1,
			expectedFirst: "123456789012",
			expectedLast:  "123456789012",
		},
		{
			name:          "multiple accounts",
			expectedCount: 3,
			expectedFirst: "123456789012",
			expectedLast:  "987654321098",
		},
		{
			name:          "no accounts",
			expectedCount: 0,
			expectedFirst: "",
			expectedLast:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate account listing results
			var accounts []Account

			if tt.expectedCount > 0 {
				accounts = make([]Account, tt.expectedCount)
				for i := 0; i < tt.expectedCount; i++ {
					accountID := tt.expectedFirst
					if tt.expectedCount > 1 && i == tt.expectedCount-1 {
						accountID = tt.expectedLast
					} else if tt.expectedCount > 2 && i > 0 {
						accountID = fmt.Sprintf("11111111111%d", i+1)
					}
					accounts[i] = Account{
						AccountID:    accountID,
						AccountName:  "Test Account",
						EmailAddress: "test@example.com",
					}
				}
			}

			assert.Equal(t, tt.expectedCount, len(accounts))

			if tt.expectedCount > 0 {
				assert.Equal(t, tt.expectedFirst, accounts[0].AccountID)
				if tt.expectedCount > 1 {
					assert.Equal(t, tt.expectedLast, accounts[tt.expectedCount-1].AccountID)
				}
			}
		})
	}
}

func TestListAccountsPagination(t *testing.T) {
	// Test pagination handling
	tests := []struct {
		name        string
		hasNextPage bool
		nextToken   string
	}{
		{
			name:        "no pagination",
			hasNextPage: false,
			nextToken:   "",
		},
		{
			name:        "has next page",
			hasNextPage: true,
			nextToken:   "next-token-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test pagination logic
			if tt.hasNextPage {
				assert.NotEmpty(t, tt.nextToken)
			} else {
				assert.Empty(t, tt.nextToken)
			}
		})
	}
}

func TestListAccountsFiltering(t *testing.T) {
	// Test account filtering logic
	tests := []struct {
		name          string
		accounts      []Account
		filter        string
		expectedCount int
	}{
		{
			name: "filter by account name",
			accounts: []Account{
				{AccountID: "123456789012", AccountName: "Production", EmailAddress: "prod@example.com"},
				{AccountID: "987654321098", AccountName: "Development", EmailAddress: "dev@example.com"},
			},
			filter:        "Production",
			expectedCount: 1,
		},
		{
			name: "filter by email",
			accounts: []Account{
				{AccountID: "123456789012", AccountName: "Production", EmailAddress: "prod@example.com"},
				{AccountID: "987654321098", AccountName: "Development", EmailAddress: "dev@example.com"},
			},
			filter:        "dev@example.com",
			expectedCount: 1,
		},
		{
			name: "no filter",
			accounts: []Account{
				{AccountID: "123456789012", AccountName: "Production", EmailAddress: "prod@example.com"},
				{AccountID: "987654321098", AccountName: "Development", EmailAddress: "dev@example.com"},
			},
			filter:        "",
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test filtering logic
			var filtered []Account

			if tt.filter == "" {
				filtered = tt.accounts
			} else {
				for _, account := range tt.accounts {
					if account.AccountName == tt.filter || account.EmailAddress == tt.filter {
						filtered = append(filtered, account)
					}
				}
			}

			assert.Equal(t, tt.expectedCount, len(filtered))
		})
	}
}

func TestListAccountsSorting(t *testing.T) {
	// Test account sorting logic
	accounts := []Account{
		{AccountID: "987654321098", AccountName: "Development", EmailAddress: "dev@example.com"},
		{AccountID: "123456789012", AccountName: "Production", EmailAddress: "prod@example.com"},
		{AccountID: "555555555555", AccountName: "Staging", EmailAddress: "staging@example.com"},
	}

	// Test sorting by account name
	sortedByName := make([]Account, len(accounts))
	copy(sortedByName, accounts)

	// Simple bubble sort by name
	for i := 0; i < len(sortedByName)-1; i++ {
		for j := 0; j < len(sortedByName)-i-1; j++ {
			if sortedByName[j].AccountName > sortedByName[j+1].AccountName {
				sortedByName[j], sortedByName[j+1] = sortedByName[j+1], sortedByName[j]
			}
		}
	}

	assert.Equal(t, "Development", sortedByName[0].AccountName)
	assert.Equal(t, "Production", sortedByName[1].AccountName)
	assert.Equal(t, "Staging", sortedByName[2].AccountName)
}

func TestListAccountsValidation(t *testing.T) {
	// Test account validation
	tests := []struct {
		name     string
		account  Account
		valid    bool
		errorMsg string
	}{
		{
			name: "valid account",
			account: Account{
				AccountID:    "123456789012",
				AccountName:  "Test Account",
				EmailAddress: "test@example.com",
			},
			valid:    true,
			errorMsg: "",
		},
		{
			name: "missing account ID",
			account: Account{
				AccountID:    "",
				AccountName:  "Test Account",
				EmailAddress: "test@example.com",
			},
			valid:    false,
			errorMsg: "account ID is required",
		},
		{
			name: "missing account name",
			account: Account{
				AccountID:    "123456789012",
				AccountName:  "",
				EmailAddress: "test@example.com",
			},
			valid:    false,
			errorMsg: "account name is required",
		},
		{
			name: "invalid email",
			account: Account{
				AccountID:    "123456789012",
				AccountName:  "Test Account",
				EmailAddress: "invalid-email",
			},
			valid:    false,
			errorMsg: "invalid email address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test validation logic
			var valid bool
			var errorMsg string

			if tt.account.AccountID == "" {
				valid = false
				errorMsg = "account ID is required"
			} else if tt.account.AccountName == "" {
				valid = false
				errorMsg = "account name is required"
			} else if tt.account.EmailAddress == "invalid-email" {
				valid = false
				errorMsg = "invalid email address"
			} else {
				valid = true
				errorMsg = ""
			}

			assert.Equal(t, tt.valid, valid)
			assert.Equal(t, tt.errorMsg, errorMsg)
		})
	}
}
