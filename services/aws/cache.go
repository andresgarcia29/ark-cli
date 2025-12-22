package services_aws

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SaveTokenToCache saves the access token in ~/.aws/sso/cache/
func (s *SSOClient) SaveTokenToCache(token *TokenResponse) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	cacheDir := filepath.Join(homeDir, ".aws", "sso", "cache")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Generate file name (SHA1 hash of the start URL)
	fileName := generateCacheFileName(s.StartURL)
	filePath := filepath.Join(cacheDir, fileName)

	// Calculate expiration time
	expiresAt := time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)

	cachedToken := CachedToken{
		StartURL:    s.StartURL,
		Region:      s.Region,
		AccessToken: token.AccessToken,
		ExpiresAt:   expiresAt.Format(time.RFC3339),
	}

	// Serialize to JSON
	data, err := json.MarshalIndent(cachedToken, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	// Save file with restrictive permissions
	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// generateCacheFileName generates the file name based on the start URL hash
func generateCacheFileName(startURL string) string {
	hash := sha1.Sum([]byte(startURL))
	return hex.EncodeToString(hash[:]) + ".json"
}

// ReadTokenFromCache reads the access token from the cache
func ReadTokenFromCache(startURL string) (*CachedToken, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	cacheDir := filepath.Join(homeDir, ".aws", "sso", "cache")
	fileName := generateCacheFileName(startURL)
	filePath := filepath.Join(cacheDir, fileName)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	var cachedToken CachedToken
	if err := json.Unmarshal(data, &cachedToken); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache file: %w", err)
	}

	// Verify if the token has expired
	expiresAt, err := time.Parse(time.RFC3339, cachedToken.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expiration time: %w", err)
	}

	if time.Now().After(expiresAt) {
		return nil, fmt.Errorf("token has expired")
	}

	return &cachedToken, nil
}
