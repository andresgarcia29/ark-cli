package services_aws

import (
	"context"
	"fmt"

	"github.com/andresgarcia29/ark-cli/lib"
	"github.com/andresgarcia29/ark-cli/logs"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
)

// ListClusters lista todos los clusters de EKS en la región configurada
func (e *EKSClient) ListClusters(ctx context.Context) ([]string, error) {
	var clusters []string
	var nextToken *string

	for {
		input := &eks.ListClustersInput{
			MaxResults: aws.Int32(100),
			NextToken:  nextToken,
		}

		output, err := e.client.ListClusters(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list EKS clusters: %w", err)
		}

		clusters = append(clusters, output.Clusters...)

		// Si no hay más páginas, terminar
		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return clusters, nil
}

// GetClustersForAccountRegion obtiene todos los clusters para una cuenta y región específica
func GetClustersForAccountRegion(ctx context.Context, profile, accountID, region string) ([]EKSCluster, error) {
	// Crear cliente EKS
	eksClient, err := NewEKSClient(ctx, region, profile)
	if err != nil {
		return nil, fmt.Errorf("failed to create EKS client: %w", err)
	}

	// Listar clusters
	clusterNames, err := eksClient.ListClusters(ctx)
	if err != nil {
		return nil, err
	}

	// Crear objetos EKSCluster
	var clusters []EKSCluster
	for _, name := range clusterNames {
		clusters = append(clusters, EKSCluster{
			Name:      name,
			Region:    region,
			AccountID: accountID,
			Profile:   profile,
		})
	}

	return clusters, nil
}

// GetClustersForAccountMultiRegion obtiene todos los clusters para una cuenta en múltiples regiones
// VERSIÓN OPTIMIZADA: Paraleliza la búsqueda en múltiples regiones simultáneamente
func GetClustersForAccountMultiRegion(ctx context.Context, profile, accountID string, regions []string) ([]EKSCluster, error) {
	logger := logs.GetLogger()

	// Si no hay regiones, retornamos lista vacía
	if len(regions) == 0 {
		return []EKSCluster{}, nil
	}

	// Si solo hay una región, no necesitamos paralelización
	if len(regions) == 1 {
		return GetClustersForAccountRegion(ctx, profile, accountID, regions[0])
	}

	logger.Infow("Escaneando regiones en paralelo",
		"total_regions", len(regions),
		"account_id", accountID)

	// Configuración para paralelización
	config := lib.ConservativeConfig()

	// Usamos nuestra función especializada para procesar regiones en paralelo
	// Esta función maneja automáticamente:
	// - Control de concurrencia (máximo 10 regiones simultáneas)
	// - Timeouts para evitar cuelgues
	// - Recolección de resultados desde channels
	// - Manejo de errores parciales
	allClusters, err := ProcessRegionsInParallel(ctx, profile, accountID, regions, config)
	if err != nil {
		return nil, fmt.Errorf("error procesando regiones para cuenta %s: %w", accountID, err)
	}

	logger.Infow("Clusters encontrados en múltiples regiones",
		"account_id", accountID,
		"total_clusters", len(allClusters),
		"regions_scanned", len(regions))

	return allClusters, nil
}

// GetClustersFromAllAccounts obtiene clusters de todas las cuentas en las regiones especificadas
// VERSIÓN OPTIMIZADA: Paraleliza el procesamiento de múltiples cuentas AWS
func GetClustersFromAllAccounts(ctx context.Context, regions []string) ([]EKSCluster, error) {
	logger := logs.GetLogger()

	// Si no se especifican regiones, usar default
	if len(regions) == 0 {
		regions = []string{"us-west-2"}
	}

	// Paso 1: Leer todos los perfiles
	logger.Info("Leyendo perfiles desde ~/.aws/config")
	allProfiles, err := ReadAllProfilesFromConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read profiles: %w", err)
	}

	// Paso 2: Seleccionar un perfil por cuenta (priorizando ReadOnly)
	selectedProfiles := SelectProfilesPerAccount(allProfiles)
	logger.Infow("Cuentas encontradas para escanear",
		"total_accounts", len(selectedProfiles))

	if len(selectedProfiles) == 0 {
		logger.Warn("No se encontraron cuentas para procesar")
		return []EKSCluster{}, nil
	}

	// Si solo hay una cuenta, no necesitamos paralelización
	if len(selectedProfiles) == 1 {
		for accountID, profile := range selectedProfiles {
			return processAccount(ctx, accountID, profile, regions)
		}
	}

	// Configuración para paralelización
	config := lib.ConservativeConfig()

	// Convertir el mapa de perfiles a una lista de IDs de cuenta
	var accountIDs []string
	profileMap := make(map[string]ProfileConfig)
	for accountID, profile := range selectedProfiles {
		accountIDs = append(accountIDs, accountID)
		profileMap[accountID] = profile
	}

	logger.Infow("Procesando cuentas en paralelo",
		"total_accounts", len(accountIDs),
		"max_workers", config.MaxWorkers)

	// Paso 3: Usar paralelización para procesar todas las cuentas
	// Esta función ejecutará el login y obtención de clusters para cada cuenta simultáneamente
	accountResults, errors := lib.ProcessAccountsInParallel(
		ctx,
		accountIDs,
		config,
		// Esta función se ejecuta para cada cuenta en paralelo
		func(ctx context.Context, accountID string) ([]EKSCluster, error) {
			// Obtenemos la información del perfil para esta cuenta
			profile, exists := profileMap[accountID]
			if !exists {
				return nil, fmt.Errorf("no se encontró perfil para cuenta %s", accountID)
			}

			// Procesamos esta cuenta (login + obtener clusters)
			return processAccount(ctx, accountID, profile, regions)
		},
	)

	// Reportar errores pero continuar con los resultados exitosos
	if len(errors) > 0 {
		logger.Warnw("Algunas cuentas tuvieron errores",
			"error_count", len(errors))
		for _, err := range errors {
			logger.Warnf("  - %v", err)
		}
	}

	// Combinar todos los clusters de todas las cuentas exitosas
	var allClusters []EKSCluster
	for accountID, clusters := range accountResults {
		allClusters = append(allClusters, clusters...)
		logger.Infow("Cuenta contribuyó con clusters",
			"account_id", accountID,
			"clusters_count", len(clusters))
	}

	logger.Infow("Procesamiento paralelo completado",
		"total_clusters", len(allClusters),
		"successful_accounts", len(accountResults),
		"failed_accounts", len(errors))

	return allClusters, nil
}

// processAccount procesa una cuenta específica: hace login y obtiene todos los clusters
// Esta función está separada para facilitar la paralelización y el testing
func processAccount(ctx context.Context, accountID string, profile ProfileConfig, regions []string) ([]EKSCluster, error) {
	logger := logs.GetLogger()

	logger.Infow("Procesando cuenta",
		"account_id", accountID,
		"profile", profile.ProfileName,
		"role", profile.RoleName)

	// Paso 1: Login con el perfil (sin set-default para evitar conflictos en paralelo)
	logger.Debugw("Realizando login",
		"profile", profile.ProfileName)
	if err := LoginWithProfile(ctx, profile.ProfileName, false); err != nil {
		return nil, fmt.Errorf("failed to login with profile %s: %w", profile.ProfileName, err)
	}
	logger.Infow("Login exitoso",
		"profile", profile.ProfileName)

	// Paso 2: Obtener clusters en todas las regiones especificadas
	// Esta función ya está paralelizada para manejar múltiples regiones simultáneamente
	logger.Debugw("Escaneando regiones",
		"regions", regions)
	clusters, err := GetClustersForAccountMultiRegion(ctx, profile.ProfileName, accountID, regions)
	if err != nil {
		return nil, fmt.Errorf("failed to get clusters for account %s: %w", accountID, err)
	}

	if len(clusters) > 0 {
		logger.Infow("Clusters encontrados",
			"account_id", accountID,
			"clusters_count", len(clusters))
	} else {
		logger.Infow("No se encontraron clusters",
			"account_id", accountID)
	}

	return clusters, nil
}
