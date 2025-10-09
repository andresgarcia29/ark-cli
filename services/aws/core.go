package services_aws

import (
	"context"
	"fmt"

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
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %w", err)
	}

	return &SSOClient{
		oidcClient: ssooidc.NewFromConfig(cfg),
		ssoClient:  sso.NewFromConfig(cfg), // Agregar esta línea
		Region:     region,
		StartURL:   startURL,
	}, nil
}

// ClientRegistration contiene la información del cliente registrado
type ClientRegistration struct {
	ClientID     string
	ClientSecret string
	ExpiresAt    int64 // Unix timestamp de cuándo expira
}

// RegisterClient registra la aplicación como cliente con AWS SSO
func (s *SSOClient) RegisterClient(ctx context.Context) (*ClientRegistration, error) {
	input := &ssooidc.RegisterClientInput{
		ClientName: aws.String("x-cli"),
		ClientType: aws.String("public"),
	}

	output, err := s.oidcClient.RegisterClient(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to register client: %w", err)
	}

	return &ClientRegistration{
		ClientID:     aws.ToString(output.ClientId),
		ClientSecret: aws.ToString(output.ClientSecret),
		ExpiresAt:    output.ClientSecretExpiresAt,
	}, nil
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

// ProfileConfig representa la configuración de un perfil de SSO
type ProfileConfig struct {
	ProfileName string
	StartURL    string
	Region      string
	AccountID   string
	RoleName    string
	SSORegion   string
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
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithSharedConfigProfile(profile),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %w", err)
	}

	return &EKSClient{
		client: eks.NewFromConfig(cfg),
		region: region,
	}, nil
}
