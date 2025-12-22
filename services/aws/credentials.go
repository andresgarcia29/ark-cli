package services_aws

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/andresgarcia29/ark-cli/logs"
)

// WriteCredentialsFile writes credentials to ~/.aws/credentials
// If setAsDefault is true, it also writes them to the [default] profile
func WriteCredentialsFile(profileName string, creds *Credentials, setAsDefault bool) error {
	logger := logs.GetLogger()
	logger.Infow("Writing credentials file", "profile", profileName, "set_as_default", setAsDefault)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Errorw("Failed to get home directory", "error", err)
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	awsDir := filepath.Join(homeDir, ".aws")
	credentialsPath := filepath.Join(awsDir, "credentials")
	logger.Debugw("Credentials file path", "path", credentialsPath)

	// Create directory if it doesn't exist
	logger.Debugw("Ensuring .aws directory exists", "path", awsDir)
	if err := os.MkdirAll(awsDir, 0700); err != nil {
		logger.Errorw("Failed to create .aws directory", "path", awsDir, "error", err)
		return fmt.Errorf("failed to create .aws directory: %w", err)
	}

	// Read existing file if it exists
	existingContent := make(map[string]map[string]string)
	if data, err := os.ReadFile(credentialsPath); err == nil {
		logger.Debug("Reading existing credentials file")
		existingContent = parseINIFile(string(data))
		logger.Debugw("Existing profiles found", "count", len(existingContent))
	} else {
		logger.Debug("No existing credentials file found, creating new one")
	}

	// Calculate expiration time
	expirationTime := time.Unix(creds.Expiration/1000, 0) // Convert from milliseconds
	logger.Debugw("Credentials expiration", "expiration_time", expirationTime.Format(time.RFC3339))

	// Update/add the specific profile
	if existingContent[profileName] == nil {
		existingContent[profileName] = make(map[string]string)
		logger.Debugw("Creating new profile section", "profile", profileName)
	} else {
		logger.Debugw("Updating existing profile", "profile", profileName)
	}
	existingContent[profileName]["aws_access_key_id"] = creds.AccessKeyID
	existingContent[profileName]["aws_secret_access_key"] = creds.SecretAccessKey
	existingContent[profileName]["aws_session_token"] = creds.SessionToken
	existingContent[profileName]["expiration"] = expirationTime.Format(time.RFC3339)

	// If required, also set as default
	if setAsDefault {
		logger.Debug("Setting credentials as default profile")
		if existingContent["default"] == nil {
			existingContent["default"] = make(map[string]string)
		}
		existingContent["default"]["aws_access_key_id"] = creds.AccessKeyID
		existingContent["default"]["aws_secret_access_key"] = creds.SecretAccessKey
		existingContent["default"]["aws_session_token"] = creds.SessionToken
		existingContent["default"]["expiration"] = expirationTime.Format(time.RFC3339)
	}

	// Generate file content
	var content strings.Builder
	logger.Debug("Generating credentials file content")

	// Write default first if it exists
	if defaultCreds, ok := existingContent["default"]; ok {
		logger.Debug("Writing default profile section")
		content.WriteString("[default]\n")
		writeCredentialSection(&content, defaultCreds)
		content.WriteString("\n")
	}

	// Write other profiles
	profileCount := 0
	for profile, creds := range existingContent {
		if profile == "default" {
			continue // Already written
		}
		profileCount++
		logger.Debugw("Writing profile section", "profile", profile)
		content.WriteString(fmt.Sprintf("[%s]\n", profile))
		writeCredentialSection(&content, creds)
		content.WriteString("\n")
	}

	logger.Debugw("Generated credentials file content", "total_profiles", profileCount+1)

	// Write file
	logger.Debugw("Writing credentials file", "path", credentialsPath)
	if err := os.WriteFile(credentialsPath, []byte(content.String()), 0600); err != nil {
		logger.Errorw("Failed to write credentials file", "path", credentialsPath, "error", err)
		return fmt.Errorf("failed to write credentials file: %w", err)
	}

	logger.Infow("Credentials file written successfully", "profile", profileName, "path", credentialsPath)
	return nil
}

// parseINIFile parses a simple INI file
func parseINIFile(content string) map[string]map[string]string {
	result := make(map[string]map[string]string)
	lines := strings.Split(content, "\n")
	var currentSection string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Ignore empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// Detect section
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = strings.Trim(line, "[]")
			result[currentSection] = make(map[string]string)
			continue
		}

		// Parse key=value
		if currentSection != "" && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				result[currentSection][key] = value
			}
		}
	}

	return result
}

// writeCredentialSection writes a credentials section
func writeCredentialSection(builder *strings.Builder, creds map[string]string) {
	// Specific order for credentials
	if val, ok := creds["aws_access_key_id"]; ok {
		fmt.Fprintf(builder, "aws_access_key_id = %s\n", val)
	}
	if val, ok := creds["aws_secret_access_key"]; ok {
		fmt.Fprintf(builder, "aws_secret_access_key = %s\n", val)
	}
	if val, ok := creds["aws_session_token"]; ok {
		fmt.Fprintf(builder, "aws_session_token = %s\n", val)
	}
	if val, ok := creds["expiration"]; ok {
		fmt.Fprintf(builder, "expiration = %s\n", val)
	}
}
