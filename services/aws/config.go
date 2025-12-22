package services_aws

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/andresgarcia29/ark-cli/logs"
)

// WriteConfigFile writes profiles to the ~/.aws/config file
func (s *SSOClient) WriteConfigFile(profiles []AWSProfile) error {
	logger := logs.GetLogger()
	logger.Infow("Writing config file", "profiles_count", len(profiles), "start_url", s.StartURL, "region", s.Region)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Errorw("Failed to get home directory", "error", err)
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".aws")
	configPath := filepath.Join(configDir, "config")
	logger.Debugw("Config file path", "path", configPath)

	// Create directory if it doesn't exist
	logger.Debugw("Ensuring .aws directory exists", "path", configDir)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		logger.Errorw("Failed to create .aws directory", "path", configDir, "error", err)
		return fmt.Errorf("failed to create .aws directory: %w", err)
	}

	// Generate file content
	var content strings.Builder
	logger.Debug("Generating config file content")

	for _, profile := range profiles {
		profileName := generateProfileName(profile.AccountName, profile.RoleName)
		logger.Debugw("Writing profile", "profile_name", profileName, "account_id", profile.AccountID, "role_name", profile.RoleName)

		content.WriteString(fmt.Sprintf("[profile %s]\n", profileName))
		content.WriteString(fmt.Sprintf("sso_start_url = %s\n", s.StartURL))
		content.WriteString(fmt.Sprintf("sso_region = %s\n", s.Region))
		content.WriteString(fmt.Sprintf("sso_account_id = %s\n", profile.AccountID))
		content.WriteString(fmt.Sprintf("sso_role_name = %s\n", profile.RoleName))
		content.WriteString(fmt.Sprintf("region = %s\n", s.Region))
		content.WriteString("\n") // Blank line between profiles
	}

	logger.Debugw("Generated config file content", "total_profiles", len(profiles))

	// Write file
	logger.Debugw("Writing config file", "path", configPath)
	if err := os.WriteFile(configPath, []byte(content.String()), 0600); err != nil {
		logger.Errorw("Failed to write config file", "path", configPath, "error", err)
		return fmt.Errorf("failed to write config file: %w", err)
	}

	logger.Infow("Config file written successfully", "path", configPath, "profiles_count", len(profiles))
	return nil
}

// generateProfileName generates a sanitized profile name
func generateProfileName(accountName, roleName string) string {
	// Convert to lowercase and replace spaces/special characters with hyphens
	name := strings.ToLower(accountName + "-" + roleName)
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")

	// Remove invalid characters (keep only letters, numbers, and hyphens)
	var result strings.Builder
	for _, char := range name {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' {
			result.WriteRune(char)
		}
	}

	return result.String()
}

// parseProfileFromConfigData parses a specific profile from configuration file data
func parseProfileFromConfigData(data []byte, profileName string) (*ProfileConfig, error) {
	lines := strings.Split(string(data), "\n")
	var currentProfile string
	profileConfig := &ProfileConfig{
		ProfileName: profileName,
	}
	found := false

	targetProfile := fmt.Sprintf("[profile %s]", profileName)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Detect profile start
		if strings.HasPrefix(line, "[profile ") {
			currentProfile = line
			if currentProfile == targetProfile {
				found = true
			}
		}

		// If we are in the correct profile, read its properties
		if found && currentProfile == targetProfile && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				switch key {
				case "sso_start_url":
					profileConfig.StartURL = value
				case "sso_region":
					profileConfig.SSORegion = value
				case "sso_account_id":
					profileConfig.AccountID = value
				case "sso_role_name":
					profileConfig.RoleName = value
				case "region":
					profileConfig.Region = value
				case "role_arn":
					profileConfig.RoleARN = value
				case "source_profile":
					profileConfig.SourceProfile = value
				case "external_id":
					profileConfig.ExternalID = value
				}
			}
		}

		// If we find another profile after ours, terminate
		if found && currentProfile != targetProfile && strings.HasPrefix(line, "[profile ") {
			break
		}
	}

	if !found {
		return nil, nil
	}

	// Determine profile type based on found properties
	if profileConfig.RoleARN != "" {
		profileConfig.ProfileType = ProfileTypeAssumeRole
	} else if profileConfig.StartURL != "" {
		profileConfig.ProfileType = ProfileTypeSSO
	} else {
		return nil, fmt.Errorf("profile %s is neither SSO nor assume role profile", profileName)
	}

	return profileConfig, nil
}

// ReadProfileFromConfig reads a specific profile from ~/.aws/config and ~/.aws/custom_config files
func ReadProfileFromConfig(profileName string) (*ProfileConfig, error) {
	logger := logs.GetLogger()
	logger.Debugw("Reading profile from config", "profile", profileName)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Errorw("Failed to get home directory", "error", err)
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	// First try to read from custom_config if it exists (has priority)
	customConfigPath := filepath.Join(homeDir, ".aws", "custom_config")
	if data, err := os.ReadFile(customConfigPath); err == nil {
		logger.Debugw("Reading from custom_config", "path", customConfigPath)
		if profileConfig, err := parseProfileFromConfigData(data, profileName); err == nil && profileConfig != nil {
			logger.Debugw("Profile found in custom_config", "profile", profileName, "type", profileConfig.ProfileType)
			return profileConfig, nil
		}
	} else if !os.IsNotExist(err) {
		logger.Warnw("Error reading custom_config (will continue with main config)", "path", customConfigPath, "error", err)
	}

	// If not found in custom_config, read from main config
	configPath := filepath.Join(homeDir, ".aws", "config")
	logger.Debugw("Reading from main config", "path", configPath)

	data, err := os.ReadFile(configPath)
	if err != nil {
		logger.Errorw("Failed to read config file", "path", configPath, "error", err)
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	profileConfig, err := parseProfileFromConfigData(data, profileName)
	if err != nil {
		logger.Errorw("Failed to parse profile", "profile", profileName, "error", err)
		return nil, err
	}

	if profileConfig == nil {
		logger.Warnw("Profile not found in config", "profile", profileName)
		return nil, fmt.Errorf("profile %s not found in config", profileName)
	}

	logger.Debugw("Profile configuration loaded successfully", "profile", profileName, "type", profileConfig.ProfileType)
	return profileConfig, nil
}

// ResolveSSOConfiguration resolves the SSO configuration for a profile
// If it's an assume role profile, it gets the configuration from the source profile
func ResolveSSOConfiguration(profileName string) (ssoRegion, ssoStartURL string, err error) {
	profileConfig, err := ReadProfileFromConfig(profileName)
	if err != nil {
		return "", "", fmt.Errorf("failed to read profile config: %w", err)
	}

	// If it's a direct SSO profile, return its configuration
	if profileConfig.ProfileType == ProfileTypeSSO {
		if profileConfig.SSORegion == "" || profileConfig.StartURL == "" {
			return "", "", fmt.Errorf("profile %s has incomplete SSO configuration (region: %s, start_url: %s)",
				profileName, profileConfig.SSORegion, profileConfig.StartURL)
		}
		return profileConfig.SSORegion, profileConfig.StartURL, nil
	}

	// If it's an assume role profile, get the configuration from the source profile
	if profileConfig.ProfileType == ProfileTypeAssumeRole {
		if profileConfig.SourceProfile == "" {
			return "", "", fmt.Errorf("assume role profile %s is missing source_profile", profileName)
		}

		sourceProfileConfig, err := ReadProfileFromConfig(profileConfig.SourceProfile)
		if err != nil {
			return "", "", fmt.Errorf("failed to read source profile %s: %w", profileConfig.SourceProfile, err)
		}

		if sourceProfileConfig.ProfileType == ProfileTypeSSO {
			if sourceProfileConfig.SSORegion == "" || sourceProfileConfig.StartURL == "" {
				return "", "", fmt.Errorf("source profile %s has incomplete SSO configuration (region: %s, start_url: %s)",
					profileConfig.SourceProfile, sourceProfileConfig.SSORegion, sourceProfileConfig.StartURL)
			}
			return sourceProfileConfig.SSORegion, sourceProfileConfig.StartURL, nil
		}

		return "", "", fmt.Errorf("source profile %s is not an SSO profile (type: %s)", profileConfig.SourceProfile, sourceProfileConfig.ProfileType)
	}

	return "", "", fmt.Errorf("profile %s does not have SSO configuration (type: %s)", profileName, profileConfig.ProfileType)
}

// parseAllProfilesFromConfigData parses all profiles from configuration file data
func parseAllProfilesFromConfigData(data []byte) ([]ProfileConfig, error) {
	var profiles []ProfileConfig
	lines := strings.Split(string(data), "\n")
	var currentProfile *ProfileConfig

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Detect profile start
		if strings.HasPrefix(line, "[profile ") && strings.HasSuffix(line, "]") {
			// Save the previous profile if it exists and is valid
			if currentProfile != nil && (currentProfile.AccountID != "" || currentProfile.RoleARN != "") {
				// Determine profile type
				if currentProfile.RoleARN != "" {
					currentProfile.ProfileType = ProfileTypeAssumeRole
				} else if currentProfile.StartURL != "" {
					currentProfile.ProfileType = ProfileTypeSSO
				}
				profiles = append(profiles, *currentProfile)
			}

			// Extract profile name
			profileName := strings.TrimSuffix(strings.TrimPrefix(line, "[profile "), "]")
			currentProfile = &ProfileConfig{
				ProfileName: profileName,
			}
		}

		// Read current profile properties
		if currentProfile != nil && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				switch key {
				case "sso_start_url":
					currentProfile.StartURL = value
				case "sso_region":
					currentProfile.SSORegion = value
				case "sso_account_id":
					currentProfile.AccountID = value
				case "sso_role_name":
					currentProfile.RoleName = value
				case "region":
					currentProfile.Region = value
				case "role_arn":
					currentProfile.RoleARN = value
				case "source_profile":
					currentProfile.SourceProfile = value
				case "external_id":
					currentProfile.ExternalID = value
				}
			}
		}
	}

	// Add the last profile if it is valid
	if currentProfile != nil && (currentProfile.AccountID != "" || currentProfile.RoleARN != "") {
		// Determine profile type
		if currentProfile.RoleARN != "" {
			currentProfile.ProfileType = ProfileTypeAssumeRole
		} else if currentProfile.StartURL != "" {
			currentProfile.ProfileType = ProfileTypeSSO
		}
		profiles = append(profiles, *currentProfile)
	}

	return profiles, nil
}

// ReadAllProfilesFromConfig reads all profiles from ~/.aws/config and ~/.aws/custom_config files
// Profiles from custom_config have priority over main config
func ReadAllProfilesFromConfig() ([]ProfileConfig, error) {
	logger := logs.GetLogger()
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Read profiles from main config file
	configPath := filepath.Join(homeDir, ".aws", "config")
	profilesMap := make(map[string]ProfileConfig)

	data, err := os.ReadFile(configPath)
	if err != nil {
		logger.Warnw("Failed to read main config file (will try custom_config)", "path", configPath, "error", err)
	} else {
		logger.Debugw("Reading profiles from main config", "path", configPath)
		profiles, err := parseAllProfilesFromConfigData(data)
		if err != nil {
			logger.Warnw("Failed to parse main config (will try custom_config)", "error", err)
		} else {
			// Add profiles from main config to map
			for _, profile := range profiles {
				profilesMap[profile.ProfileName] = profile
			}
			logger.Debugw("Loaded profiles from main config", "count", len(profiles))
		}
	}

	// Read profiles from custom_config file if it exists (has priority)
	customConfigPath := filepath.Join(homeDir, ".aws", "custom_config")
	if data, err := os.ReadFile(customConfigPath); err == nil {
		logger.Debugw("Reading profiles from custom_config", "path", customConfigPath)
		customProfiles, err := parseAllProfilesFromConfigData(data)
		if err != nil {
			logger.Warnw("Failed to parse custom_config", "error", err)
		} else {
			// Profiles from custom_config overwrite or add to main config profiles
			for _, profile := range customProfiles {
				profilesMap[profile.ProfileName] = profile
			}
			logger.Debugw("Merged profiles from custom_config", "count", len(customProfiles), "total", len(profilesMap))
		}
	} else if !os.IsNotExist(err) {
		logger.Warnw("Error reading custom_config (will continue with main config only)", "path", customConfigPath, "error", err)
	}

	// Convert map to slice
	var profiles []ProfileConfig
	for _, profile := range profilesMap {
		profiles = append(profiles, profile)
	}

	logger.Debugw("Total profiles loaded", "count", len(profiles))
	return profiles, nil
}

// SelectProfilesPerAccount selects one profile per account, prioritizing ReadOnlyAccess
func SelectProfilesPerAccount(profiles []ProfileConfig, prefixs []string) map[string]ProfileConfig {
	accountProfiles := make(map[string][]ProfileConfig)

	// Group profiles by account
	for _, profile := range profiles {
		accountProfiles[profile.AccountID] = append(accountProfiles[profile.AccountID], profile)
	}

	// Select the best profile per account
	selectedProfiles := make(map[string]ProfileConfig)

	for accountID, accountProfileList := range accountProfiles {
		var selected ProfileConfig
		foundReadOnly := false

		// Search for ReadOnlyAccess first
		for _, profile := range accountProfileList {
			roleName := strings.ToLower(profile.RoleName)
			found := slices.ContainsFunc(prefixs, func(p string) bool {
				return strings.Contains(roleName, p)
			})
			if found {
				fmt.Println("profile found", profile)
				selected = profile
				foundReadOnly = true
				break
			}
		}

		// If ReadOnly wasn't found, use the first one
		if !foundReadOnly && len(accountProfileList) > 0 {
			selected = accountProfileList[0]
		}

		selectedProfiles[accountID] = selected
	}

	return selectedProfiles
}

// SelectProfileByARN selects a profile matching the provided role ARN
func SelectProfileByARN(profiles []ProfileConfig, roleARN string) map[string]ProfileConfig {
	selectedProfiles := make(map[string]ProfileConfig)

	for _, profile := range profiles {
		if profile.RoleARN == roleARN {
			selectedProfiles[profile.AccountID] = profile
			// We only need one profile for this ARN/account
			break
		}
	}

	return selectedProfiles
}
