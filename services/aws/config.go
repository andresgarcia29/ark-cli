package services_aws

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/andresgarcia29/ark-cli/logs"
)

// arkManagedKeys are the keys ark writes for an SSO profile. A profile whose
// keys are all ark-managed is safe to replace; anything else is merged.
var arkManagedKeys = []string{"sso_start_url", "sso_region", "sso_account_id", "sso_role_name", "region"}

// WriteConfigFile merges the discovered SSO profiles into ~/.aws/config.
// Profiles the user wrote by hand, including assume-role profiles that ark
// itself resolves through source_profile, are preserved. The write is atomic.
func (s *SSOClient) WriteConfigFile(profiles []AWSProfile) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	configPath := filepath.Join(home, ".aws", "config")

	fileLock.Lock()
	defer fileLock.Unlock()

	file, err := readINI(configPath)
	if err != nil {
		return err
	}

	added := 0
	for _, profile := range profiles {
		name := "profile " + generateProfileName(profile.AccountName, profile.RoleName)
		if !file.has(name) {
			added++
		}
		section := file.section(name)
		section.set("sso_start_url", s.StartURL)
		section.set("sso_region", s.Region)
		section.set("sso_account_id", profile.AccountID)
		section.set("sso_role_name", profile.RoleName)
		if section.Value["region"] == "" {
			section.set("region", s.Region)
		}
	}

	file.sortSections()
	if err := writeFileAtomic(configPath, file.render(), 0600); err != nil {
		return err
	}

	logs.GetLogger().Debugw("config merged", "path", configPath, "written", len(profiles), "new", added)
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
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	profileConfig, err := parseProfileFromConfigData(data, profileName)
	if err != nil {
		return nil, err
	}

	if profileConfig == nil {
		logger.Warnw("Profile not found in config", "profile", profileName)
		return nil, fmt.Errorf("no profile named %q in ~/.aws/config", profileName)
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
