package services_aws

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// WriteConfigFile escribe los perfiles al archivo ~/.aws/config
func (s *SSOClient) WriteConfigFile(profiles []AWSProfile) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".aws")
	configPath := filepath.Join(configDir, "config")

	// Crear directorio si no existe
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create .aws directory: %w", err)
	}

	// Generar contenido del archivo
	var content strings.Builder

	for _, profile := range profiles {
		profileName := generateProfileName(profile.AccountName, profile.RoleName)

		content.WriteString(fmt.Sprintf("[profile %s]\n", profileName))
		content.WriteString(fmt.Sprintf("sso_start_url = %s\n", s.StartURL))
		content.WriteString(fmt.Sprintf("sso_region = %s\n", s.Region))
		content.WriteString(fmt.Sprintf("sso_account_id = %s\n", profile.AccountID))
		content.WriteString(fmt.Sprintf("sso_role_name = %s\n", profile.RoleName))
		content.WriteString(fmt.Sprintf("region = %s\n", s.Region))
		content.WriteString("\n") // Línea en blanco entre perfiles
	}

	// Escribir archivo
	if err := os.WriteFile(configPath, []byte(content.String()), 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// generateProfileName genera un nombre de perfil sanitizado
func generateProfileName(accountName, roleName string) string {
	// Convertir a minúsculas y reemplazar espacios/caracteres especiales con guiones
	name := strings.ToLower(accountName + "-" + roleName)
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")

	// Remover caracteres no válidos (mantener solo letras, números y guiones)
	var result strings.Builder
	for _, char := range name {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' {
			result.WriteRune(char)
		}
	}

	return result.String()
}

// ReadProfileFromConfig lee un perfil específico del archivo ~/.aws/config
func ReadProfileFromConfig(profileName string) (*ProfileConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".aws", "config")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parsear el archivo INI manualmente
	lines := strings.Split(string(data), "\n")
	var currentProfile string
	profileConfig := &ProfileConfig{
		ProfileName: profileName,
	}
	found := false

	targetProfile := fmt.Sprintf("[profile %s]", profileName)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Detectar inicio de perfil
		if strings.HasPrefix(line, "[profile ") {
			currentProfile = line
			if currentProfile == targetProfile {
				found = true
			}
		}

		// Si estamos en el perfil correcto, leer sus propiedades
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

		// Si encontramos otro perfil después del nuestro, terminar
		if found && currentProfile != targetProfile && strings.HasPrefix(line, "[profile ") {
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("profile %s not found in config", profileName)
	}

	// Determinar el tipo de perfil basado en las propiedades encontradas
	if profileConfig.RoleARN != "" {
		profileConfig.ProfileType = ProfileTypeAssumeRole
	} else if profileConfig.StartURL != "" {
		profileConfig.ProfileType = ProfileTypeSSO
	} else {
		return nil, fmt.Errorf("profile %s is neither SSO nor assume role profile", profileName)
	}

	return profileConfig, nil
}

// ResolveSSOConfiguration resuelve la configuración SSO para un perfil
// Si es un perfil assume role, obtiene la configuración del source profile
func ResolveSSOConfiguration(profileName string) (ssoRegion, ssoStartURL string, err error) {
	profileConfig, err := ReadProfileFromConfig(profileName)
	if err != nil {
		return "", "", fmt.Errorf("failed to read profile config: %w", err)
	}

	// Si es un perfil SSO directo, devolver su configuración
	if profileConfig.ProfileType == ProfileTypeSSO {
		if profileConfig.SSORegion == "" || profileConfig.StartURL == "" {
			return "", "", fmt.Errorf("profile %s has incomplete SSO configuration (region: %s, start_url: %s)",
				profileName, profileConfig.SSORegion, profileConfig.StartURL)
		}
		return profileConfig.SSORegion, profileConfig.StartURL, nil
	}

	// Si es un perfil assume role, obtener la configuración del source profile
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

// ReadAllProfilesFromConfig lee todos los perfiles del archivo ~/.aws/config
func ReadAllProfilesFromConfig() ([]ProfileConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".aws", "config")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var profiles []ProfileConfig
	lines := strings.Split(string(data), "\n")
	var currentProfile *ProfileConfig

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Detectar inicio de perfil
		if strings.HasPrefix(line, "[profile ") && strings.HasSuffix(line, "]") {
			// Guardar el perfil anterior si existe y es válido
			if currentProfile != nil && (currentProfile.AccountID != "" || currentProfile.RoleARN != "") {
				// Determinar el tipo de perfil
				if currentProfile.RoleARN != "" {
					currentProfile.ProfileType = ProfileTypeAssumeRole
				} else if currentProfile.StartURL != "" {
					currentProfile.ProfileType = ProfileTypeSSO
				}
				profiles = append(profiles, *currentProfile)
			}

			// Extraer nombre del perfil
			profileName := strings.TrimSuffix(strings.TrimPrefix(line, "[profile "), "]")
			currentProfile = &ProfileConfig{
				ProfileName: profileName,
			}
		}

		// Leer propiedades del perfil actual
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

	// Agregar el último perfil si es válido
	if currentProfile != nil && (currentProfile.AccountID != "" || currentProfile.RoleARN != "") {
		// Determinar el tipo de perfil
		if currentProfile.RoleARN != "" {
			currentProfile.ProfileType = ProfileTypeAssumeRole
		} else if currentProfile.StartURL != "" {
			currentProfile.ProfileType = ProfileTypeSSO
		}
		profiles = append(profiles, *currentProfile)
	}

	return profiles, nil
}

// SelectProfilesPerAccount selecciona un perfil por cuenta, priorizando ReadOnlyAccess
func SelectProfilesPerAccount(profiles []ProfileConfig, prefixs []string) map[string]ProfileConfig {
	accountProfiles := make(map[string][]ProfileConfig)

	// Agrupar perfiles por cuenta
	for _, profile := range profiles {
		accountProfiles[profile.AccountID] = append(accountProfiles[profile.AccountID], profile)
	}

	// Seleccionar el mejor perfil por cuenta
	selectedProfiles := make(map[string]ProfileConfig)

	for accountID, accountProfileList := range accountProfiles {
		var selected ProfileConfig
		foundReadOnly := false

		// Buscar ReadOnlyAccess primero
		for _, profile := range accountProfileList {
			roleName := strings.ToLower(profile.RoleName)
			if slices.Contains(prefixs, roleName) {
				selected = profile
				foundReadOnly = true
				break
			}
		}

		// Si no se encontró ReadOnly, usar el primero
		if !foundReadOnly && len(accountProfileList) > 0 {
			selected = accountProfileList[0]
		}

		selectedProfiles[accountID] = selected
	}

	return selectedProfiles
}
