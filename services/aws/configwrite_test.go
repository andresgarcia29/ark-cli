package services_aws

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// WriteConfigFile used to overwrite ~/.aws/config wholesale, destroying
// hand-written profiles including the assume-role ones ark itself resolves.
func TestWriteConfigFilePreservesHandWrittenProfiles(t *testing.T) {
	home := withHome(t)
	path := filepath.Join(home, ".aws", "config")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}

	handWritten := `[default]
region = us-east-1

[profile prod-admin]
role_arn = arn:aws:iam::111122223333:role/Admin
source_profile = acme-readonly
external_id = shared-secret
`
	if err := os.WriteFile(path, []byte(handWritten), 0600); err != nil {
		t.Fatal(err)
	}

	client := &SSOClient{Region: "us-east-1", StartURL: "https://acme.awsapps.com/start"}
	err := client.WriteConfigFile([]AWSProfile{
		{AccountID: "111122223333", AccountName: "Acme", RoleName: "ReadOnlyAccess"},
	})
	if err != nil {
		t.Fatalf("WriteConfigFile: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"[profile prod-admin]",
		"role_arn = arn:aws:iam::111122223333:role/Admin",
		"source_profile = acme-readonly",
		"external_id = shared-secret",
		"region = us-east-1",
	} {
		if !strings.Contains(string(got), want) {
			t.Errorf("WriteConfigFile destroyed %q:\n%s", want, got)
		}
	}
	if !strings.Contains(string(got), "[profile acme-readonlyaccess]") {
		t.Errorf("new SSO profile was not written:\n%s", got)
	}
}

func TestWriteConfigFileKeepsUserRegionOverride(t *testing.T) {
	home := withHome(t)
	path := filepath.Join(home, ".aws", "config")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	// The user pinned this profile to a different region than the SSO home.
	seed := "[profile acme-readonlyaccess]\nregion = ap-southeast-2\n"
	if err := os.WriteFile(path, []byte(seed), 0600); err != nil {
		t.Fatal(err)
	}

	client := &SSOClient{Region: "us-east-1", StartURL: "https://acme.awsapps.com/start"}
	if err := client.WriteConfigFile([]AWSProfile{
		{AccountID: "111122223333", AccountName: "Acme", RoleName: "ReadOnlyAccess"},
	}); err != nil {
		t.Fatal(err)
	}

	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "region = ap-southeast-2") {
		t.Errorf("user's region override was overwritten:\n%s", got)
	}
	if !strings.Contains(string(got), "sso_account_id = 111122223333") {
		t.Errorf("SSO settings were not refreshed:\n%s", got)
	}
}

func TestGenerateProfileName(t *testing.T) {
	cases := map[string]string{
		"Acme Corp/ReadOnlyAccess": "acme-corp-readonlyaccess",
		"acme_prod/Admin":          "acme-prod-admin",
		"Acme (EU)/Dev":            "acme-eu-dev",
	}
	for input, want := range cases {
		account, role, _ := strings.Cut(input, "/")
		if got := generateProfileName(account, role); got != want {
			t.Errorf("generateProfileName(%q, %q) = %q, want %q", account, role, got, want)
		}
	}
}
