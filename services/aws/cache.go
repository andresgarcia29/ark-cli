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

// SaveTokenToCache guarda el access token en ~/.aws/sso/cache/
func (s *SSOClient) SaveTokenToCache(token *TokenResponse) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	cacheDir := filepath.Join(homeDir, ".aws", "sso", "cache")

	// Crear directorio si no existe
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Generar nombre del archivo (hash SHA1 del start URL)
	fileName := generateCacheFileName(s.StartURL)
	filePath := filepath.Join(cacheDir, fileName)

	// Calcular tiempo de expiración
	expiresAt := time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)

	cachedToken := CachedToken{
		StartURL:    s.StartURL,
		Region:      s.Region,
		AccessToken: token.AccessToken,
		ExpiresAt:   expiresAt.Format(time.RFC3339),
	}

	// Serializar a JSON
	data, err := json.MarshalIndent(cachedToken, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	// Guardar archivo con permisos restrictivos
	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// generateCacheFileName genera el nombre del archivo basado en el hash del start URL
func generateCacheFileName(startURL string) string {
	hash := sha1.Sum([]byte(startURL))
	return hex.EncodeToString(hash[:]) + ".json"
}

// ReadTokenFromCache lee el access token del cache
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

	// Verificar si el token ha expirado
	expiresAt, err := time.Parse(time.RFC3339, cachedToken.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expiration time: %w", err)
	}

	if time.Now().After(expiresAt) {
		return nil, fmt.Errorf("token has expired")
	}

	return &cachedToken, nil
}
