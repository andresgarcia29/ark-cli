package services_aws

import (
	"context"
	"fmt"

	"github.com/andresgarcia29/ark-cli/logs"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
)

type SSOClient struct {
	oidcClient *ssooidc.Client
	ssoClient  *sso.Client
	Region     string
	StartURL   string
}

func NewSSOClient(ctx context.Context, region, startURL string) (*SSOClient, error) {
	logger := logs.GetLogger()
	logger.Debugw("Creating new SSO client", "region", region, "start_url", startURL)

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		logger.Errorw("Failed to load SDK config", "region", region, "error", err)
		return nil, fmt.Errorf("unable to load SDK config: %w", err)
	}

	client := &SSOClient{
		oidcClient: ssooidc.NewFromConfig(cfg),
		ssoClient:  sso.NewFromConfig(cfg), // Agregar esta línea
		Region:     region,
		StartURL:   startURL,
	}

	logger.Debugw("SSO client created successfully", "region", region, "start_url", startURL)
	return client, nil
}

// ClientRegistration contiene la información del cliente registrado
type ClientRegistration struct {
	ClientID     string
	ClientSecret string
	ExpiresAt    int64 // Unix timestamp de cuándo expira
}

// RegisterClient registra la aplicación como cliente con AWS SSO
func (s *SSOClient) RegisterClient(ctx context.Context) (*ClientRegistration, error) {
	logger := logs.GetLogger()
	logger.Debug("Registering client with AWS SSO")

	input := &ssooidc.RegisterClientInput{
		ClientName: aws.String("x-cli"),
		ClientType: aws.String("public"),
	}

	output, err := s.oidcClient.RegisterClient(ctx, input)
	if err != nil {
		logger.Errorw("Failed to register client", "error", err)
		return nil, fmt.Errorf("failed to register client: %w", err)
	}

	registration := &ClientRegistration{
		ClientID:     aws.ToString(output.ClientId),
		ClientSecret: aws.ToString(output.ClientSecret),
		ExpiresAt:    output.ClientSecretExpiresAt,
	}

	logger.Debugw("Client registered successfully", "client_id", registration.ClientID, "expires_at", registration.ExpiresAt)
	return registration, nil
}

// DeviceAuthorization contiene la información de autorización del dispositivo
type DeviceAuthorization struct {
	DeviceCode              string
	UserCode                string
	VerificationURI         string
	VerificationURIComplete string
	ExpiresIn               int32
	Interval                int32 // Segundos entre cada polling
}

// TokenResponse contiene el access token y metadata
type TokenResponse struct {
	AccessToken  string
	ExpiresIn    int32
	TokenType    string
	RefreshToken string
}

type CachedToken struct {
	StartURL    string `json:"startUrl"`
	Region      string `json:"region"`
	AccessToken string `json:"accessToken"`
	ExpiresAt   string `json:"expiresAt"` // ISO8601 format
}

// Account representa una cuenta de AWS
type Account struct {
	AccountID    string
	AccountName  string
	EmailAddress string
}

// Role representa un rol en una cuenta
type Role struct {
	RoleName  string
	AccountID string
}

// AWSProfile representa una combinación de cuenta y rol
type AWSProfile struct {
	AccountID    string
	AccountName  string
	RoleName     string
	EmailAddress string
}

// ProfileType representa el tipo de perfil
type ProfileType string

const (
	ProfileTypeSSO        ProfileType = "sso"
	ProfileTypeAssumeRole ProfileType = "assume_role"
)

// ProfileConfig representa la configuración de un perfil de AWS
type ProfileConfig struct {
	ProfileName string
	ProfileType ProfileType
	StartURL    string
	Region      string
	AccountID   string
	RoleName    string
	SSORegion   string
	// Campos para assume role
	RoleARN       string
	SourceProfile string
	ExternalID    string
}

// Credentials representa las credenciales temporales de AWS
type Credentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Expiration      int64 // Unix timestamp
}

// EKSCluster representa un cluster de EKS
type EKSCluster struct {
	Name      string
	Region    string
	AccountID string
	Profile   string
}

// EKSClient encapsula el cliente de EKS
type EKSClient struct {
	client *eks.Client
	region string
}

// NewEKSClient crea una nueva instancia de EKSClient
func NewEKSClient(ctx context.Context, region, profile string) (*EKSClient, error) {
	logger := logs.GetLogger()
	logger.Debugw("Creating new EKS client", "region", region, "profile", profile)

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithSharedConfigProfile(profile),
	)
	if err != nil {
		logger.Errorw("Failed to load SDK config for EKS client", "region", region, "profile", profile, "error", err)
		return nil, fmt.Errorf("unable to load SDK config: %w", err)
	}

	client := &EKSClient{
		client: eks.NewFromConfig(cfg),
		region: region,
	}

	logger.Debugw("EKS client created successfully", "region", region, "profile", profile)
	return client, nil
}
