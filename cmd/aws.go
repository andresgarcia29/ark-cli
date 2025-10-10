package cmd

import (
	"context"
	"fmt"

	controllers "github.com/andresgarcia29/ark-cli/controllers/aws"
	animation "github.com/andresgarcia29/ark-cli/lib/animation"
	"github.com/spf13/cobra"
)

var (
	awsCmd = &cobra.Command{
		Use:   "aws",
		Short: "AWS related operations",
		Long:  `AWS related operations - Interactive profile selection and login`,
		Run:   aws,
	}
)

func init() {
	rootCmd.AddCommand(awsCmd)
}

func aws(cmd *cobra.Command, args []string) {
	// Crear contexto
	ctx := context.Background()

	// Mostrar selector interactivo de perfiles
	selectedProfile, err := animation.InteractiveProfileSelector()
	if err != nil {
		fmt.Printf("❌ Error selecting profile: %v\n", err)
		return
	}

	// Mostrar información del perfil seleccionado
	fmt.Printf("\n✅ Selected profile: %s (%s)\n", selectedProfile.ProfileName, selectedProfile.ProfileType)
	fmt.Println("🔐 Logging in...")

	// Realizar login con el perfil seleccionado usando retry
	if err := controllers.AttemptLoginWithRetry(ctx, selectedProfile.ProfileName, true, selectedProfile.SSORegion, selectedProfile.StartURL); err != nil {
		fmt.Printf("❌ Login failed after retry: %v\n", err)
		return
	}

	fmt.Printf("🎉 Successfully logged in with profile: %s\n", selectedProfile.ProfileName)
	fmt.Println("💡 You can now use AWS CLI commands with this profile")
}
