package services_aws

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

const sampleConfig = `[default]
region = us-east-1

[profile acme-readonly]
sso_start_url = https://acme.awsapps.com/start
sso_region = us-east-1
sso_account_id = 111122223333
sso_role_name = ReadOnlyAccess
region = us-west-2

[profile acme-admin]
role_arn = arn:aws:iam::111122223333:role/Admin
source_profile = acme-readonly
external_id = secret
region = us-west-2

[profile orphan]
region = eu-west-1
`

func seedConfig(t *testing.T, body string) string {
	t.Helper()
	home := withHome(t)
	path := filepath.Join(home, ".aws", "config")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseProfileFromConfigData(t *testing.T) {
	sso, err := parseProfileFromConfigData([]byte(sampleConfig), "acme-readonly")
	if err != nil {
		t.Fatalf("parse sso profile: %v", err)
	}
	if sso.ProfileType != ProfileTypeSSO {
		t.Errorf("type = %q, want sso", sso.ProfileType)
	}
	if sso.AccountID != "111122223333" || sso.RoleName != "ReadOnlyAccess" {
		t.Errorf("wrong sso fields: %+v", sso)
	}
	if sso.Region != "us-west-2" {
		t.Errorf("region = %q, want us-west-2", sso.Region)
	}

	assume, err := parseProfileFromConfigData([]byte(sampleConfig), "acme-admin")
	if err != nil {
		t.Fatalf("parse assume-role profile: %v", err)
	}
	if assume.ProfileType != ProfileTypeAssumeRole {
		t.Errorf("type = %q, want assume_role", assume.ProfileType)
	}
	if assume.SourceProfile != "acme-readonly" || assume.ExternalID != "secret" {
		t.Errorf("wrong assume-role fields: %+v", assume)
	}

	// A profile that is neither SSO nor assume-role is an error, not a panic.
	if _, err := parseProfileFromConfigData([]byte(sampleConfig), "orphan"); err == nil {
		t.Error("orphan profile should not parse as a usable profile")
	}

	// Absent profiles report nothing rather than failing.
	got, err := parseProfileFromConfigData([]byte(sampleConfig), "missing")
	if err != nil || got != nil {
		t.Errorf("missing profile: got %+v, %v", got, err)
	}
}

// A profile's keys must not bleed into the profile declared after it.
func TestParseProfileDoesNotLeakAcrossSections(t *testing.T) {
	got, err := parseProfileFromConfigData([]byte(sampleConfig), "acme-readonly")
	if err != nil {
		t.Fatal(err)
	}
	if got.RoleARN != "" || got.SourceProfile != "" {
		t.Errorf("assume-role keys leaked into the SSO profile: %+v", got)
	}
}

func TestParseAllProfilesFromConfigData(t *testing.T) {
	profiles, err := parseAllProfilesFromConfigData([]byte(sampleConfig))
	if err != nil {
		t.Fatal(err)
	}

	byName := map[string]ProfileConfig{}
	for _, p := range profiles {
		byName[p.ProfileName] = p
	}
	for _, want := range []string{"acme-readonly", "acme-admin"} {
		if _, ok := byName[want]; !ok {
			t.Errorf("profile %q missing from %v", want, byName)
		}
	}
}

func TestResolveSSOConfiguration(t *testing.T) {
	seedConfig(t, sampleConfig)

	region, url, err := ResolveSSOConfiguration("acme-readonly")
	if err != nil {
		t.Fatalf("direct SSO profile: %v", err)
	}
	if region != "us-east-1" || url != "https://acme.awsapps.com/start" {
		t.Errorf("got %q / %q", region, url)
	}

	// An assume-role profile must inherit SSO settings from its source.
	region, url, err = ResolveSSOConfiguration("acme-admin")
	if err != nil {
		t.Fatalf("assume-role profile: %v", err)
	}
	if region != "us-east-1" || url != "https://acme.awsapps.com/start" {
		t.Errorf("source profile not followed: %q / %q", region, url)
	}

	if _, _, err := ResolveSSOConfiguration("nope"); err == nil {
		t.Error("unknown profile should error")
	}
}

func TestResolveSSOConfigurationRejectsBrokenChain(t *testing.T) {
	seedConfig(t, `[profile dangling]
role_arn = arn:aws:iam::1:role/X
source_profile = gone
`)
	if _, _, err := ResolveSSOConfiguration("dangling"); err == nil {
		t.Error("a source_profile that does not exist should error")
	}
}

func TestTokenCacheRoundTrip(t *testing.T) {
	withHome(t)
	client := &SSOClient{Region: "us-east-1", StartURL: "https://acme.awsapps.com/start"}

	if err := client.SaveTokenToCache(&TokenResponse{AccessToken: "tok", ExpiresIn: 3600}); err != nil {
		t.Fatalf("SaveTokenToCache: %v", err)
	}

	got, err := ReadTokenFromCache(client.StartURL)
	if err != nil {
		t.Fatalf("ReadTokenFromCache: %v", err)
	}
	if got.AccessToken != "tok" {
		t.Errorf("token = %q, want tok", got.AccessToken)
	}

	// The cache file is a credential and must not be world readable.
	path := filepath.Join(os.Getenv("HOME"), ".aws", "sso", "cache", generateCacheFileName(client.StartURL))
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("cache permissions = %o, want 600", perm)
	}
}

func TestReadTokenFromCacheRejectsExpired(t *testing.T) {
	withHome(t)
	client := &SSOClient{Region: "us-east-1", StartURL: "https://acme.awsapps.com/start"}

	if err := client.SaveTokenToCache(&TokenResponse{AccessToken: "tok", ExpiresIn: -1}); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadTokenFromCache(client.StartURL); err == nil {
		t.Error("an expired token must not be returned")
	}
}

func TestReadTokenFromCacheMissing(t *testing.T) {
	withHome(t)
	if _, err := ReadTokenFromCache("https://absent.awsapps.com/start"); err == nil {
		t.Error("missing cache file should error")
	}
}

// The cache key must match the AWS CLI's scheme so both tools share sessions.
func TestGenerateCacheFileNameIsStable(t *testing.T) {
	const url = "https://acme.awsapps.com/start"
	first := generateCacheFileName(url)
	if first != generateCacheFileName(url) {
		t.Error("cache file name is not deterministic")
	}
	if first == generateCacheFileName(url+"/other") {
		t.Error("different start URLs must not collide")
	}
	if filepath.Ext(first) != ".json" {
		t.Errorf("cache file %q should end in .json", first)
	}
}

func TestCachedCredentialsSkewBoundary(t *testing.T) {
	withHome(t)
	creds := &Credentials{
		AccessKeyID: "AKIA", SecretAccessKey: "s", SessionToken: "t",
		Expiration: time.Now().Add(30 * time.Minute).UnixMilli(),
	}
	if err := WriteCredentialsFile("p", creds, false); err != nil {
		t.Fatal(err)
	}

	if !CachedCredentialsValid("p", 5*time.Minute) {
		t.Error("30 minutes left should satisfy a 5 minute skew")
	}
	if CachedCredentialsValid("p", time.Hour) {
		t.Error("30 minutes left must not satisfy a 1 hour skew")
	}
}
