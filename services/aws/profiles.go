package services_aws

import (
	"context"
	"fmt"

	"github.com/andresgarcia29/ark-cli/lib"
	"github.com/andresgarcia29/ark-cli/logs"
)

// GetAllProfiles obtiene todas las combinaciones de cuenta+rol disponibles
// VERSIÓN OPTIMIZADA: Paraleliza la obtención de roles para múltiples cuentas
func (s *SSOClient) GetAllProfiles(ctx context.Context, accessToken string) ([]AWSProfile, error) {
	logger := logs.GetLogger()

	// Paso 1: Obtener todas las cuentas (esto debe ser secuencial)
	logger.Info("Obteniendo lista de cuentas")
	accounts, err := s.ListAccounts(ctx, accessToken)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo cuentas: %w", err)
	}

	logger.Infow("Cuentas encontradas, obteniendo roles en paralelo",
		"total_accounts", len(accounts))

	// Configuración para operaciones paralelas
	config := lib.ConservativeConfig()

	// Paso 2: Usar la función genérica para procesar cuentas en paralelo
	// Esta función ejecutará ListAccountRoles para cada cuenta simultáneamente
	accountRoles, errors := lib.ProcessAccountsInParallel(
		ctx,
		// Convertimos la lista de cuentas a una lista de IDs
		func() []string {
			var accountIDs []string
			for _, account := range accounts {
				accountIDs = append(accountIDs, account.AccountID)
			}
			return accountIDs
		}(),
		config,
		// Esta función se ejecuta para cada cuenta en paralelo
		func(ctx context.Context, accountID string) ([]Role, error) {
			logger.Debugf("Obteniendo roles para cuenta: %s", accountID)

			// Aquí es donde hacemos la llamada real a la API de AWS SSO
			// Esta función puede tardar varios segundos, por eso la paralelizamos
			roles, err := s.ListAccountRoles(ctx, accessToken, accountID)
			if err != nil {
				return nil, fmt.Errorf("error obteniendo roles para cuenta %s: %w", accountID, err)
			}

			logger.Infow("Roles obtenidos para cuenta",
				"account_id", accountID,
				"roles_count", len(roles))
			return roles, nil
		},
	)

	// Si hubo errores en algunas cuentas, los reportamos pero continuamos
	if len(errors) > 0 {
		logger.Warnw("Algunas cuentas tuvieron errores",
			"error_count", len(errors))
		for _, err := range errors {
			logger.Warnf("  - %v", err)
		}
	}

	// Paso 3: Convertir los resultados a perfiles
	// Necesitamos combinar la información de accounts con los roles obtenidos
	var profiles []AWSProfile

	// Creamos un mapa para búsqueda rápida de información de cuentas
	accountMap := make(map[string]Account)
	for _, account := range accounts {
		accountMap[account.AccountID] = account
	}

	// Para cada cuenta que fue procesada exitosamente
	for accountID, roles := range accountRoles {
		// Buscamos la información completa de la cuenta
		account, found := accountMap[accountID]
		if !found {
			// Esto no debería pasar, pero lo manejamos por seguridad
			logger.Warnw("No se encontró información completa para cuenta",
				"account_id", accountID)
			continue
		}

		// Creamos un perfil por cada combinación cuenta+rol
		for _, role := range roles {
			profiles = append(profiles, AWSProfile{
				AccountID:    account.AccountID,
				AccountName:  account.AccountName,
				RoleName:     role.RoleName,
				EmailAddress: account.EmailAddress,
			})
		}
	}

	logger.Infow("Perfiles creados exitosamente",
		"total_profiles", len(profiles))
	return profiles, nil
}

// LoginWithProfile realiza el login completo con un perfil específico
func LoginWithProfile(ctx context.Context, profileName string, setAsDefault bool) error {
	// Paso 1: Leer configuración del perfil
	profileConfig, err := ReadProfileFromConfig(profileName)
	if err != nil {
		return fmt.Errorf("failed to read profile config: %w", err)
	}

	// Paso 2: Leer token del cache
	cachedToken, err := ReadTokenFromCache(profileConfig.StartURL)
	if err != nil {
		return fmt.Errorf("failed to read token from cache (you may need to run login first): %w", err)
	}

	// Paso 3: Crear cliente SSO
	client, err := NewSSOClient(ctx, profileConfig.SSORegion, profileConfig.StartURL)
	if err != nil {
		return fmt.Errorf("failed to create SSO client: %w", err)
	}

	// Paso 4: Obtener credenciales temporales
	creds, err := client.GetRoleCredentials(ctx, cachedToken.AccessToken, profileConfig.AccountID, profileConfig.RoleName)
	if err != nil {
		return fmt.Errorf("failed to get role credentials: %w", err)
	}

	// Paso 5: Escribir credenciales al archivo
	if err := WriteCredentialsFile(profileName, creds, setAsDefault); err != nil {
		return fmt.Errorf("failed to write credentials: %w", err)
	}

	return nil
}
