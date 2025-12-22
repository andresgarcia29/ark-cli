package services_aws

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSelectProfilesPerAccount(t *testing.T) {
	tests := []struct {
		name     string
		profiles []ProfileConfig
		prefixs  []string
		expected map[string]ProfileConfig
	}{
		{
			name: "single account with ReadOnlyAccess",
			profiles: []ProfileConfig{
				{
					AccountID:   "123456789012",
					ProfileName: "account1-readonlyaccess",
					RoleName:    "ReadOnlyAccess",
				},
			},
			prefixs: []string{"readonlyaccess"},
			expected: map[string]ProfileConfig{
				"123456789012": {
					AccountID:   "123456789012",
					ProfileName: "account1-readonlyaccess",
					RoleName:    "ReadOnlyAccess",
				},
			},
		},
		{
			name: "single account with multiple roles prioritizes ReadOnlyAccess",
			profiles: []ProfileConfig{
				{
					AccountID:   "123456789012",
					ProfileName: "account1-admin",
					RoleName:    "AdministratorAccess",
				},
				{
					AccountID:   "123456789012",
					ProfileName: "account1-readonlyaccess",
					RoleName:    "ReadOnlyAccess",
				},
				{
					AccountID:   "123456789012",
					ProfileName: "account1-developer",
					RoleName:    "DeveloperAccess",
				},
			},
			prefixs: []string{"readonlyaccess"},
			expected: map[string]ProfileConfig{
				"123456789012": {
					AccountID:   "123456789012",
					ProfileName: "account1-readonlyaccess",
					RoleName:    "ReadOnlyAccess",
				},
			},
		},
		{
			name: "single account without ReadOnlyAccess uses first profile",
			profiles: []ProfileConfig{
				{
					AccountID:   "123456789012",
					ProfileName: "account1-admin",
					RoleName:    "AdministratorAccess",
				},
				{
					AccountID:   "123456789012",
					ProfileName: "account1-developer",
					RoleName:    "DeveloperAccess",
				},
			},
			prefixs: []string{"readonlyaccess"},
			expected: map[string]ProfileConfig{
				"123456789012": {
					AccountID:   "123456789012",
					ProfileName: "account1-admin",
					RoleName:    "AdministratorAccess",
				},
			},
		},
		{
			name: "multiple accounts with different roles",
			profiles: []ProfileConfig{
				{
					AccountID:   "123456789012",
					ProfileName: "account1-readonlyaccess",
					RoleName:    "ReadOnlyAccess",
				},
				{
					AccountID:   "123456789012",
					ProfileName: "account1-admin",
					RoleName:    "AdministratorAccess",
				},
				{
					AccountID:   "987654321098",
					ProfileName: "account2-developer",
					RoleName:    "DeveloperAccess",
				},
				{
					AccountID:   "987654321098",
					ProfileName: "account2-readonlyaccess",
					RoleName:    "ReadOnlyAccess",
				},
			},
			prefixs: []string{"readonlyaccess"},
			expected: map[string]ProfileConfig{
				"123456789012": {
					AccountID:   "123456789012",
					ProfileName: "account1-readonlyaccess",
					RoleName:    "ReadOnlyAccess",
				},
				"987654321098": {
					AccountID:   "987654321098",
					ProfileName: "account2-readonlyaccess",
					RoleName:    "ReadOnlyAccess",
				},
			},
		},
		{
			name: "case insensitive role name matching",
			profiles: []ProfileConfig{
				{
					AccountID:   "123456789012",
					ProfileName: "account1-readonly",
					RoleName:    "ReadOnlyAccess",
				},
			},
			prefixs: []string{"readonlyaccess"},
			expected: map[string]ProfileConfig{
				"123456789012": {
					AccountID:   "123456789012",
					ProfileName: "account1-readonly",
					RoleName:    "ReadOnlyAccess",
				},
			},
		},
		{
			name: "multiple priority prefixes",
			profiles: []ProfileConfig{
				{
					AccountID:   "123456789012",
					ProfileName: "account1-viewonly",
					RoleName:    "ViewOnlyAccess",
				},
				{
					AccountID:   "123456789012",
					ProfileName: "account1-admin",
					RoleName:    "AdministratorAccess",
				},
			},
			prefixs: []string{"readonlyaccess", "viewonlyaccess"},
			expected: map[string]ProfileConfig{
				"123456789012": {
					AccountID:   "123456789012",
					ProfileName: "account1-viewonly",
					RoleName:    "ViewOnlyAccess",
				},
			},
		},
		{
			name:     "empty profiles list",
			profiles: []ProfileConfig{},
			prefixs:  []string{"readonlyaccess"},
			expected: map[string]ProfileConfig{},
		},
		{
			name: "empty prefixes list uses first profile",
			profiles: []ProfileConfig{
				{
					AccountID:   "123456789012",
					ProfileName: "account1-admin",
					RoleName:    "AdministratorAccess",
				},
			},
			prefixs: []string{},
			expected: map[string]ProfileConfig{
				"123456789012": {
					AccountID:   "123456789012",
					ProfileName: "account1-admin",
					RoleName:    "AdministratorAccess",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SelectProfilesPerAccount(tt.profiles, tt.prefixs)

			assert.Equal(t, len(tt.expected), len(result), "Number of selected profiles should match")

			for accountID, expectedProfile := range tt.expected {
				actualProfile, exists := result[accountID]
				assert.True(t, exists, "Account %s should exist in result", accountID)
				assert.Equal(t, expectedProfile.AccountID, actualProfile.AccountID)
				assert.Equal(t, expectedProfile.ProfileName, actualProfile.ProfileName)
				assert.Equal(t, expectedProfile.RoleName, actualProfile.RoleName)
			}
		})
	}
}

func TestSelectProfileByARN(t *testing.T) {
	profiles := []ProfileConfig{
		{
			AccountID:   "123456789012",
			ProfileName: "account1-role1",
			RoleARN:     "arn:aws:iam::123456789012:role/Role1",
		},
		{
			AccountID:   "123456789012",
			ProfileName: "account1-role2",
			RoleARN:     "arn:aws:iam::123456789012:role/Role2",
		},
		{
			AccountID:   "987654321098",
			ProfileName: "account2-role1",
			RoleARN:     "arn:aws:iam::987654321098:role/Role1",
		},
	}

	tests := []struct {
		name     string
		roleARN  string
		expected map[string]ProfileConfig
	}{
		{
			name:    "match existing ARN",
			roleARN: "arn:aws:iam::123456789012:role/Role1",
			expected: map[string]ProfileConfig{
				"123456789012": {
					AccountID:   "123456789012",
					ProfileName: "account1-role1",
					RoleARN:     "arn:aws:iam::123456789012:role/Role1",
				},
			},
		},
		{
			name:    "match another account ARN",
			roleARN: "arn:aws:iam::987654321098:role/Role1",
			expected: map[string]ProfileConfig{
				"987654321098": {
					AccountID:   "987654321098",
					ProfileName: "account2-role1",
					RoleARN:     "arn:aws:iam::987654321098:role/Role1",
				},
			},
		},
		{
			name:     "no match",
			roleARN:  "arn:aws:iam::123456789012:role/NonExistent",
			expected: map[string]ProfileConfig{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SelectProfileByARN(profiles, tt.roleARN)

			assert.Equal(t, len(tt.expected), len(result))
			for accountID, expectedProfile := range tt.expected {
				actualProfile, exists := result[accountID]
				assert.True(t, exists)
				assert.Equal(t, expectedProfile.RoleARN, actualProfile.RoleARN)
			}
		})
	}
}
