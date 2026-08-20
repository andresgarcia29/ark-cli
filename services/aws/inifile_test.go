package services_aws

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// withHome points os.UserHomeDir at a temp directory so tests never touch the
// developer's real ~/.aws.
func withHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
}

func TestParseINIKeepsUnknownKeysAndOrder(t *testing.T) {
	f := parseINI([]byte(`
# a comment
[default]
region = us-east-1
credential_process = /usr/bin/creds

[profile work]
sso_start_url = https://example.awsapps.com/start
region = eu-west-1
`))

	if !f.has("default") || !f.has("profile work") {
		t.Fatalf("missing sections: %v", f.order)
	}
	if got := f.section("default").Value["credential_process"]; got != "/usr/bin/creds" {
		t.Errorf("credential_process = %q, want it preserved", got)
	}
	if got := f.section("profile work").Value["region"]; got != "eu-west-1" {
		t.Errorf("region = %q, want eu-west-1", got)
	}

	// A round trip must not lose the unmanaged key.
	if !strings.Contains(string(f.render()), "credential_process = /usr/bin/creds") {
		t.Error("render dropped an unmanaged key")
	}
}

func TestWriteFileAtomicSetsPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "creds")
	if err := writeFileAtomic(path, []byte("data"), 0600); err != nil {
		t.Fatalf("writeFileAtomic: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("permissions = %o, want 600", perm)
	}
	// No temp files may survive a successful write.
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Errorf("leftover files: %d", len(entries))
	}
}

func TestWriteCredentialsFilePreservesOtherProfiles(t *testing.T) {
	home := withHome(t)
	path := filepath.Join(home, ".aws", "credentials")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	existing := "[other]\naws_access_key_id = KEEP\nregion = sa-east-1\n"
	if err := os.WriteFile(path, []byte(existing), 0600); err != nil {
		t.Fatal(err)
	}

	creds := &Credentials{
		AccessKeyID:     "AKIANEW",
		SecretAccessKey: "secret",
		SessionToken:    "token",
		Expiration:      time.Now().Add(time.Hour).UnixMilli(),
	}
	if err := WriteCredentialsFile("mine", creds, false); err != nil {
		t.Fatalf("WriteCredentialsFile: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"[other]", "KEEP", "region = sa-east-1", "[mine]", "AKIANEW"} {
		if !strings.Contains(string(got), want) {
			t.Errorf("credentials file lost %q:\n%s", want, got)
		}
	}
}

func TestWriteCredentialsFileSetAsDefault(t *testing.T) {
	withHome(t)
	creds := &Credentials{
		AccessKeyID:     "AKIA",
		SecretAccessKey: "s",
		SessionToken:    "t",
		Expiration:      time.Now().Add(time.Hour).UnixMilli(),
	}
	if err := WriteCredentialsFile("mine", creds, true); err != nil {
		t.Fatal(err)
	}

	path, _ := credentialsPath()
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "[default]") {
		t.Errorf("default profile not written:\n%s", data)
	}
	// default must sort first so the file reads the way the AWS CLI writes it.
	if idx := strings.Index(string(data), "[default]"); idx != 0 {
		t.Errorf("[default] should lead the file, found at %d", idx)
	}
}

// Parallel logins race on ~/.aws/credentials; the write must stay serialized.
func TestWriteCredentialsFileConcurrent(t *testing.T) {
	withHome(t)
	creds := func(id string) *Credentials {
		return &Credentials{
			AccessKeyID:     id,
			SecretAccessKey: "s",
			SessionToken:    "t",
			Expiration:      time.Now().Add(time.Hour).UnixMilli(),
		}
	}

	names := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	var wg sync.WaitGroup
	for _, name := range names {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := WriteCredentialsFile(name, creds("AKIA"+name), false); err != nil {
				t.Errorf("write %s: %v", name, err)
			}
		}()
	}
	wg.Wait()

	path, _ := credentialsPath()
	data, _ := os.ReadFile(path)
	for _, name := range names {
		if !strings.Contains(string(data), "["+name+"]") {
			t.Errorf("profile %s was lost to a concurrent write:\n%s", name, data)
		}
	}
}

func TestCachedCredentialsValid(t *testing.T) {
	withHome(t)
	fresh := &Credentials{
		AccessKeyID: "AKIA", SecretAccessKey: "s", SessionToken: "t",
		Expiration: time.Now().Add(time.Hour).UnixMilli(),
	}
	if err := WriteCredentialsFile("fresh", fresh, false); err != nil {
		t.Fatal(err)
	}
	stale := &Credentials{
		AccessKeyID: "AKIA", SecretAccessKey: "s", SessionToken: "t",
		Expiration: time.Now().Add(-time.Hour).UnixMilli(),
	}
	if err := WriteCredentialsFile("stale", stale, false); err != nil {
		t.Fatal(err)
	}

	if !CachedCredentialsValid("fresh", time.Minute) {
		t.Error("fresh credentials should be reusable")
	}
	if CachedCredentialsValid("stale", time.Minute) {
		t.Error("expired credentials must not be reused")
	}
	if CachedCredentialsValid("absent", time.Minute) {
		t.Error("missing profile must not be reported as cached")
	}
	// The skew must push a nearly-expired credential over the line.
	if CachedCredentialsValid("fresh", 2*time.Hour) {
		t.Error("skew larger than the remaining lifetime should invalidate")
	}
}
