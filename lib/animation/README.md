# AWS Menu Package

This package contains all the display and user interface logic for AWS profile selection.

## Structure

- `selector.go`: Implementation of the interactive profile selector using Bubble Tea

## Responsibilities

- **User Interface**: Handling the interactive interface with Bubble Tea
- **Display**: Formatting and presentation of AWS profiles
- **Navigation**: Keyboard control and profile selection
- **Styles**: Application of colors and styles with Lip Gloss

## Separation of Responsibilities

This package focuses solely on **presentation** and user **interaction**. The business logic remains in `services/aws`:

- `services/aws`: Business logic, configuration, authentication
- `lib/animation`: User interface, display, interaction

## Usage

```go
import "github.com/andresgarcia29/ark-cli/lib/animation"

// Show interactive selector
selectedProfile, err := animation.InteractiveProfileSelector()
if err != nil {
    // Handle error
}

// Use selected profile
fmt.Printf("Selected: %s\n", selectedProfile.ProfileName)
```

## Features

- **Bubble Tea**: Modern framework for TUI
- **Lip Gloss**: Terminal styles and colors
- **Intuitive Navigation**: Arrows, vim keys (j/k)
- **Differentiated Colors**: SSO (green), Assume Role (orange)
- **Detailed Information**: Account, role, region
