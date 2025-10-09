# AWS Menu Package

Este paquete contiene toda la lógica de visualización e interfaz de usuario para la selección de perfiles de AWS.

## Estructura

- `selector.go`: Implementación del selector interactivo de perfiles usando Bubble Tea

## Responsabilidades

- **Interfaz de Usuario**: Manejo de la interfaz interactiva con Bubble Tea
- **Visualización**: Formateo y presentación de perfiles de AWS
- **Navegación**: Control de teclado y selección de perfiles
- **Estilos**: Aplicación de colores y estilos con Lip Gloss

## Separación de Responsabilidades

Este paquete se enfoca únicamente en la **presentación** y **interacción** del usuario. La lógica de negocio permanece en `services/aws`:

- `services/aws`: Lógica de negocio, configuración, autenticación
- `lib/menu/aws`: Interfaz de usuario, visualización, interacción

## Uso

```go
import aws_menu "github.com/andresgarcia29/ark-cli/lib/menu/aws"

// Mostrar selector interactivo
selectedProfile, err := aws_menu.InteractiveProfileSelector()
if err != nil {
    // Manejar error
}

// Usar el perfil seleccionado
fmt.Printf("Selected: %s\n", selectedProfile.ProfileName)
```

## Características

- **Bubble Tea**: Framework moderno para TUI
- **Lip Gloss**: Estilos y colores para terminal
- **Navegación intuitiva**: Flechas, vim keys (j/k)
- **Colores diferenciados**: SSO (verde), Assume Role (naranja)
- **Información detallada**: Cuenta, rol, región
