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
		ssoClient:  sso.NewFromConfig(cfg), // Add this line
		Region:     region,
		StartURL:   startURL,
	}

	logger.Debugw("SSO client created successfully", "region", region, "start_url", startURL)
	return client, nil
}

// ClientRegistration contains registered client information
type ClientRegistration struct {
	ClientID     string
	ClientSecret string
	ExpiresAt    int64 // Unix timestamp of when it expires
}

// RegisterClient registers the application as a client with AWS SSO
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

// DeviceAuthorization contains device authorization information
type DeviceAuthorization struct {
	DeviceCode              string
	UserCode                string
	VerificationURI         string
	VerificationURIComplete string
	ExpiresIn               int32
	Interval                int32 // Seconds between each polling
}

// TokenResponse contains access token and metadata
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

// Account represents an AWS account
type Account struct {
	AccountID    string
	AccountName  string
	EmailAddress string
}

// Role represents a role in an account
type Role struct {
	RoleName  string
	AccountID string
}

// AWSProfile represents a combination of account and role
type AWSProfile struct {
	AccountID    string
	AccountName  string
	RoleName     string
	EmailAddress string
}

// ProfileType represents the profile type
type ProfileType string

const (
	ProfileTypeSSO        ProfileType = "sso"
	ProfileTypeAssumeRole ProfileType = "assume_role"
)

// ProfileConfig represents the configuration of an AWS profile
type ProfileConfig struct {
	ProfileName string
	ProfileType ProfileType
	StartURL    string
	Region      string
	AccountID   string
	RoleName    string
	SSORegion   string
	// Assume role fields
	RoleARN       string
	SourceProfile string
	ExternalID    string
}

// Credentials represents temporary AWS credentials
type Credentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Expiration      int64 // Unix timestamp
}

// EKSCluster represents an EKS cluster
type EKSCluster struct {
	Name      string
	Region    string
	AccountID string
	Profile   string
}

// EKSClient encapsulates the EKS client
type EKSClient struct {
	client *eks.Client
	region string
}

// NewEKSClient creates a new instance of EKSClient
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
