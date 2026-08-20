package services_aws

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// credentialKeys are the keys ark manages; every other key in a profile is
// left untouched so user settings such as region survive a login.
var credentialKeys = []string{
	"aws_access_key_id",
	"aws_secret_access_key",
	"aws_session_token",
	"expiration",
}

func credentialsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".aws", "credentials"), nil
}

// WriteCredentialsFile merges creds into ~/.aws/credentials under profileName,
// optionally mirroring them into [default]. Other profiles and unmanaged keys
// are preserved; the write is atomic and serialized against concurrent logins.
func WriteCredentialsFile(profileName string, creds *Credentials, setAsDefault bool) error {
	if creds == nil {
		return fmt.Errorf("no credentials to write for profile %s", profileName)
	}

	path, err := credentialsPath()
	if err != nil {
		return err
	}

	fileLock.Lock()
	defer fileLock.Unlock()

	file, err := readINI(path)
	if err != nil {
		return err
	}

	expiry := time.UnixMilli(creds.Expiration).UTC().Format(time.RFC3339)
	apply := func(section string) {
		s := file.section(section)
		s.set("aws_access_key_id", creds.AccessKeyID)
		s.set("aws_secret_access_key", creds.SecretAccessKey)
		s.set("aws_session_token", creds.SessionToken)
		s.set("expiration", expiry)
	}

	apply(profileName)
	if setAsDefault {
		apply("default")
	}

	file.sortSections()
	if err := writeFileAtomic(path, file.render(), 0600); err != nil {
		return err
	}
	return nil
}

// CachedCredentialsValid reports whether ~/.aws/credentials already holds
// unexpired credentials for profileName, letting a repeat login skip AWS.
func CachedCredentialsValid(profileName string, within time.Duration) bool {
	path, err := credentialsPath()
	if err != nil {
		return false
	}

	fileLock.Lock()
	defer fileLock.Unlock()

	file, err := readINI(path)
	if err != nil || !file.has(profileName) {
		return false
	}

	s := file.section(profileName)
	for _, k := range credentialKeys {
		if s.Value[k] == "" {
			return false
		}
	}

	expiry, err := time.Parse(time.RFC3339, s.Value["expiration"])
	if err != nil {
		return false
	}
	return time.Now().Add(within).Before(expiry)
}
