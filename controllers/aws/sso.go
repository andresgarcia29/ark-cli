package controllers

import (
	"context"
	"fmt"
	"strings"

	services_aws "github.com/andresgarcia29/ark-cli/services/aws"
)

func AWSSSOLogin(ctx context.Context, SSORegion string, SSOStartURL string, boostraping bool) error {
	// Paso 1: Crear el cliente SSO
	client, err := services_aws.NewSSOClient(ctx, SSORegion, SSOStartURL)
	if err != nil {
		fmt.Println("Error creating SSO client:", err)
		return err
	}
	fmt.Printf("SSO client created successfully for region: %s, start URL: %s\n", client.Region, client.StartURL)

	// Paso 2: Registrar el cliente
	fmt.Println("\nRegistering client...")
	registration, err := client.RegisterClient(ctx)
	if err != nil {
		fmt.Println("Error registering client:", err)
		return err
	}
	fmt.Println("Client registered successfully")

	// Paso 3: Iniciar autorización del dispositivo
	fmt.Println("\nStarting device authorization...")
	deviceAuth, err := client.StartDeviceAuthorization(ctx, registration.ClientID, registration.ClientSecret)
	if err != nil {
		fmt.Println("Error starting device authorization:", err)
		return err
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
		return err
	}
	fmt.Println("\n✓ Authorization successful!")

	// Paso 6: Guardar token en cache
	fmt.Println("Saving token to cache...")
	if err := client.SaveTokenToCache(token); err != nil {
		fmt.Println("Error saving token:", err)
		return err
	}
	fmt.Println("✓ Token saved successfully")

	if boostraping {
		// Paso 7: Obtener todas las cuentas y roles
		fmt.Println("\nFetching accounts and roles...")
		profiles, err := client.GetAllProfiles(ctx, token.AccessToken)
		if err != nil {
			fmt.Println("Error getting profiles:", err)
			return err
		}
		fmt.Printf("✓ Found %d profiles\n", len(profiles))

		// Paso 8: Escribir el archivo config
		fmt.Println("\nWriting profiles to ~/.aws/config...")
		if err := client.WriteConfigFile(profiles); err != nil {
			fmt.Println("Error writing config file:", err)
			return err
		}
		fmt.Println("✓ Config file updated successfully")
	}

	fmt.Println("\n🎉 AWS SSO sso completed!")

	return nil
}
