package cmd

import (
	"context"
	"fmt"
	"strings"

	services_aws "github.com/andresgarcia29/ark-cli/services/aws"
	"github.com/spf13/cobra"
)

var (
	SSORegion   string
	SSOStartURL string

	awsSSOnCmd = &cobra.Command{
		Use:   "sso",
		Short: "Start a new AWS SSO session",
		Long:  "Configure and start a new AWS SSO session with the provided profile, fetching the credentials from the AWS SSO cache",
		Run:   awsSSOCommand,
	}
)

func init() {
	awsCmd.AddCommand(awsSSOnCmd)
	awsSSOnCmd.Flags().StringVar(&SSORegion, "region", "us-east-1", "AWS SSO region")
	awsSSOnCmd.Flags().StringVar(&SSOStartURL, "start-url", "", "AWS SSO start URL (required)")
	awsSSOnCmd.MarkFlagRequired("start-url")
}

func awsSSOCommand(cmd *cobra.Command, args []string) {
	fmt.Println("AWS sso")
	ctx := context.Background()

	// Paso 1: Crear el cliente SSO
	client, err := services_aws.NewSSOClient(ctx, SSORegion, SSOStartURL)
	if err != nil {
		fmt.Println("Error creating SSO client:", err)
		return
	}
	fmt.Printf("SSO client created successfully for region: %s, start URL: %s\n", client.Region, client.StartURL)

	// Paso 2: Registrar el cliente
	fmt.Println("\nRegistering client...")
	registration, err := client.RegisterClient(ctx)
	if err != nil {
		fmt.Println("Error registering client:", err)
		return
	}
	fmt.Println("Client registered successfully")

	// Paso 3: Iniciar autorización del dispositivo
	fmt.Println("\nStarting device authorization...")
	deviceAuth, err := client.StartDeviceAuthorization(ctx, registration.ClientID, registration.ClientSecret)
	if err != nil {
		fmt.Println("Error starting device authorization:", err)
		return
	}

	// Paso 4: Mostrar instrucciones al usuario
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Please authorize this application:")
	fmt.Printf("Visit: %s\n", deviceAuth.VerificationURIComplete)
	fmt.Printf("Or go to: %s and enter code: %s\n", deviceAuth.VerificationURI, deviceAuth.UserCode)
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("\nWaiting for authorization...")

	// Paso 5: Polling para obtener el token
	token, err := client.CreateToken(ctx, registration.ClientID, registration.ClientSecret, deviceAuth.DeviceCode, deviceAuth.Interval)
	if err != nil {
		fmt.Println("Error creating token:", err)
		return
	}
	fmt.Println("\n✓ Authorization successful!")

	// Paso 6: Guardar token en cache
	fmt.Println("Saving token to cache...")
	if err := client.SaveTokenToCache(token); err != nil {
		fmt.Println("Error saving token:", err)
		return
	}
	fmt.Println("✓ Token saved successfully")

	// Paso 7: Obtener todas las cuentas y roles
	fmt.Println("\nFetching accounts and roles...")
	profiles, err := client.GetAllProfiles(ctx, token.AccessToken)
	if err != nil {
		fmt.Println("Error getting profiles:", err)
		return
	}
	fmt.Printf("✓ Found %d profiles\n", len(profiles))

	// Paso 8: Escribir el archivo config
	fmt.Println("\nWriting profiles to ~/.aws/config...")
	if err := client.WriteConfigFile(profiles); err != nil {
		fmt.Println("Error writing config file:", err)
		return
	}
	fmt.Println("✓ Config file updated successfully")

	fmt.Println("\n🎉 AWS SSO sso completed!")
}
