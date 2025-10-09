package services_aws

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// WriteCredentialsFile escribe las credenciales en ~/.aws/credentials
// Si setAsDefault es true, también las escribe en el perfil [default]
func WriteCredentialsFile(profileName string, creds *Credentials, setAsDefault bool) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	awsDir := filepath.Join(homeDir, ".aws")
	credentialsPath := filepath.Join(awsDir, "credentials")

	// Crear directorio si no existe
	if err := os.MkdirAll(awsDir, 0700); err != nil {
		return fmt.Errorf("failed to create .aws directory: %w", err)
	}

	// Leer archivo existente si existe
	existingContent := make(map[string]map[string]string)
	if data, err := os.ReadFile(credentialsPath); err == nil {
		existingContent = parseINIFile(string(data))
	}

	// Calcular tiempo de expiración
	expirationTime := time.Unix(creds.Expiration/1000, 0) // Convertir de milisegundos

	// Actualizar/agregar el perfil específico
	if existingContent[profileName] == nil {
		existingContent[profileName] = make(map[string]string)
	}
	existingContent[profileName]["aws_access_key_id"] = creds.AccessKeyID
	existingContent[profileName]["aws_secret_access_key"] = creds.SecretAccessKey
	existingContent[profileName]["aws_session_token"] = creds.SessionToken
	existingContent[profileName]["expiration"] = expirationTime.Format(time.RFC3339)

	// Si se requiere, también establecer como default
	if setAsDefault {
		if existingContent["default"] == nil {
			existingContent["default"] = make(map[string]string)
		}
		existingContent["default"]["aws_access_key_id"] = creds.AccessKeyID
		existingContent["default"]["aws_secret_access_key"] = creds.SecretAccessKey
		existingContent["default"]["aws_session_token"] = creds.SessionToken
		existingContent["default"]["expiration"] = expirationTime.Format(time.RFC3339)
	}

	// Generar contenido del archivo
	var content strings.Builder

	// Escribir default primero si existe
	if defaultCreds, ok := existingContent["default"]; ok {
		content.WriteString("[default]\n")
		writeCredentialSection(&content, defaultCreds)
		content.WriteString("\n")
	}

	// Escribir otros perfiles
	for profile, creds := range existingContent {
		if profile == "default" {
			continue // Ya lo escribimos
		}
		content.WriteString(fmt.Sprintf("[%s]\n", profile))
		writeCredentialSection(&content, creds)
		content.WriteString("\n")
	}

	// Escribir archivo
	if err := os.WriteFile(credentialsPath, []byte(content.String()), 0600); err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}

	return nil
}

// parseINIFile parsea un archivo INI simple
func parseINIFile(content string) map[string]map[string]string {
	result := make(map[string]map[string]string)
	lines := strings.Split(content, "\n")
	var currentSection string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Ignorar líneas vacías y comentarios
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// Detectar sección
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = strings.Trim(line, "[]")
			result[currentSection] = make(map[string]string)
			continue
		}

		// Parsear clave=valor
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

// writeCredentialSection escribe una sección de credenciales
func writeCredentialSection(builder *strings.Builder, creds map[string]string) {
	// Orden específico para las credenciales
	if val, ok := creds["aws_access_key_id"]; ok {
		builder.WriteString(fmt.Sprintf("aws_access_key_id = %s\n", val))
	}
	if val, ok := creds["aws_secret_access_key"]; ok {
		builder.WriteString(fmt.Sprintf("aws_secret_access_key = %s\n", val))
	}
	if val, ok := creds["aws_session_token"]; ok {
		builder.WriteString(fmt.Sprintf("aws_session_token = %s\n", val))
	}
	if val, ok := creds["expiration"]; ok {
		builder.WriteString(fmt.Sprintf("expiration = %s\n", val))
	}
}
