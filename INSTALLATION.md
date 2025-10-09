# Installation Guide

This guide provides detailed instructions for installing `ark-cli` on different platforms.

## Table of Contents

- [Homebrew (macOS/Linux)](#homebrew-macoslinux)
- [Go Install](#go-install)
- [Download Binary](#download-binary)
- [Troubleshooting](#troubleshooting)

---

## Homebrew (macOS/Linux)

### Installation

```bash
# Add the tap
brew tap andresgarcia29/agm

# Install ark-cli
brew install ark --cask
```

### Verify Installation

```bash
ark version
```

### Troubleshooting Homebrew Installation

#### Issue: "No Cask with this name exists"

If you see this error:
```
Warning: Cask 'ark' is unavailable: No Cask with this name exists.
```

**Solution:** Your local tap cache might be outdated. Update it:

```bash
# Update the tap to fetch the latest formulas/casks
brew tap andresgarcia29/agm --force
brew update

# Then install
brew install ark --cask
```

#### Issue: Command not found after installation

If `ark` command is not found after installation:

**Solution 1:** Reinstall the cask
```bash
brew reinstall ark --cask
```

**Solution 2:** Check if the binary is linked correctly
```bash
# Check if ark is in Homebrew's bin directory
ls -la /opt/homebrew/bin/ark  # For Apple Silicon
ls -la /usr/local/bin/ark      # For Intel Macs

# If missing, try reinstalling
brew uninstall ark --cask
brew install ark --cask
```

**Solution 3:** Verify your PATH includes Homebrew's bin directory
```bash
echo $PATH | grep -q homebrew && echo "Homebrew is in PATH" || echo "Homebrew NOT in PATH"

# If not in PATH, add to your shell config (~/.zshrc or ~/.bashrc):
# For Apple Silicon:
export PATH="/opt/homebrew/bin:$PATH"

# For Intel Macs:
export PATH="/usr/local/bin:$PATH"
```

#### Issue: Old tap version installed

If you have an old version of the tap:

```bash
# Check current tap info
brew tap-info andresgarcia29/agm

# Update the tap
cd /opt/homebrew/Library/Taps/andresgarcia29/homebrew-agm  # Apple Silicon
# OR
cd /usr/local/Library/Taps/andresgarcia29/homebrew-agm     # Intel

git pull

# Then reinstall
brew reinstall ark --cask
```

### Uninstall

```bash
brew uninstall ark --cask
brew untap andresgarcia29/agm
```

---

## Go Install

If you have Go installed (version 1.21 or later):

```bash
go install github.com/andresgarcia29/ark-cli@latest
```

Make sure your `$GOPATH/bin` is in your PATH:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

---

## Download Binary

### Manual Installation

1. Go to the [releases page](https://github.com/andresgarcia29/ark-cli/releases)
2. Download the appropriate binary for your platform:
   - `ark-cli_<version>_darwin_amd64.tar.gz` - macOS Intel
   - `ark-cli_<version>_darwin_arm64.tar.gz` - macOS Apple Silicon
   - `ark-cli_<version>_linux_amd64.tar.gz` - Linux x64
   - `ark-cli_<version>_linux_arm64.tar.gz` - Linux ARM64
   - `ark-cli_<version>_windows_amd64.zip` - Windows x64

3. Extract the archive:
   ```bash
   # macOS/Linux
   tar -xzf ark-cli_<version>_<platform>.tar.gz

   # Windows
   # Use your preferred extraction tool
   ```

4. Move the binary to a directory in your PATH:
   ```bash
   # macOS/Linux
   sudo mv ark /usr/local/bin/

   # Or to a user directory
   mv ark ~/bin/
   ```

5. Make it executable (macOS/Linux only):
   ```bash
   chmod +x /usr/local/bin/ark
   ```

6. Verify installation:
   ```bash
   ark version
   ```

---

## Troubleshooting

### General Issues

#### "Permission denied" error

If you get permission errors:

```bash
# macOS/Linux: Make the binary executable
chmod +x /path/to/ark

# If installing to system directory, use sudo
sudo mv ark /usr/local/bin/
```

#### Version mismatch

Check which version is installed:

```bash
ark version
which ark  # Shows which binary is being used
```

If you have multiple installations, make sure the correct one is in your PATH first.

#### macOS Gatekeeper Warning

On macOS, if you download the binary directly, you might see a security warning:

```
"ark" cannot be opened because it is from an unidentified developer
```

**Solution:**
```bash
# Remove the quarantine attribute
xattr -d com.apple.quarantine /usr/local/bin/ark

# Or allow it in System Preferences:
# System Preferences > Security & Privacy > General > "Allow anyway"
```

---

## Upgrading

### Homebrew

```bash
brew upgrade ark --cask
```

### Go Install

```bash
go install github.com/andresgarcia29/ark-cli@latest
```

### Manual

Download and install the latest binary following the [Manual Installation](#manual-installation) steps above.

---

## Support

If you encounter any issues not covered in this guide:

1. Check the [GitHub Issues](https://github.com/andresgarcia29/ark-cli/issues)
2. Create a new issue with:
   - Your OS and version
   - Installation method used
   - Complete error message
   - Output of `ark version` (if available)
