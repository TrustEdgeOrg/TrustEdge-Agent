//go:build darwin

package security

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/constants"
)

func TestParseKextstat(t *testing.T) {
	sample := `
Index Refs Address            Size       Wired      Name (Version) UUID <Linked Against>
  123    0 0xffffff7f9c000000 0x5000     0x5000     com.apple.driver.AppleHDA (300.0) <1 2 3>
   45    1 0xffffff7f9d000000 0x2000     0x2000     com.example.kext (1.2.3)
`
	got := parseKextstat(sample)
	if len(got) != 2 {
		t.Fatalf("got=%d want 2: %+v", len(got), got)
	}
	if got[0].Type != constants.TypeDriverLoad {
		t.Fatalf("type=%s", got[0].Type)
	}
	if got[0].Payload["name"] != "com.apple.driver.AppleHDA" {
		t.Fatalf("payload=%v", got[0].Payload)
	}
	if got[1].Payload["name"] != "com.example.kext" {
		t.Fatalf("payload=%v", got[1].Payload)
	}
}

func TestLaunchPlistArtifactPersistence(t *testing.T) {
	orig := plutilFn
	defer func() { plutilFn = orig }()
	plutilFn = func(path string) ([]byte, error) {
		return []byte(`{"Label":"com.trustedge.test","ProgramArguments":["/usr/bin/true","--once"]}`), nil
	}

	artifact, ok, err := launchPlistArtifact(launchScope{
		Hive:      "user",
		KeyPath:   "Library/LaunchAgents",
		Dir:       "/tmp",
		EventType: constants.TypeRegistryPersist,
	}, "/tmp/com.trustedge.test.plist")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if artifact.Type != constants.TypeRegistryPersist {
		t.Fatalf("type=%s", artifact.Type)
	}
	if artifact.Payload["value_name"] != "com.trustedge.test" {
		t.Fatalf("payload=%v", artifact.Payload)
	}
	if artifact.Payload["value"] != "/usr/bin/true --once" {
		t.Fatalf("value=%v", artifact.Payload["value"])
	}
}

func TestLaunchPlistArtifactService(t *testing.T) {
	orig := plutilFn
	defer func() { plutilFn = orig }()
	plutilFn = func(path string) ([]byte, error) {
		return []byte(`{"Label":"com.trustedge.daemon","Program":"/usr/local/bin/daemon"}`), nil
	}

	artifact, ok, err := launchPlistArtifact(launchScope{
		Hive:      "system",
		KeyPath:   "Library/LaunchDaemons",
		Dir:       "/Library/LaunchDaemons",
		EventType: constants.TypeServiceInstall,
	}, "/Library/LaunchDaemons/com.trustedge.daemon.plist")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if artifact.Type != constants.TypeServiceInstall {
		t.Fatalf("type=%s", artifact.Type)
	}
	if artifact.Payload["name"] != "com.trustedge.daemon" {
		t.Fatalf("payload=%v", artifact.Payload)
	}
}

func TestCollectLaunchPlistsFromDir(t *testing.T) {
	dir := t.TempDir()
	plistPath := filepath.Join(dir, "com.trustedge.agent.plist")
	if err := os.WriteFile(plistPath, []byte(`ignored`), 0o644); err != nil {
		t.Fatal(err)
	}

	origPlutil := plutilFn
	origReadDir := readDirFn
	defer func() {
		plutilFn = origPlutil
		readDirFn = origReadDir
	}()
	readDirFn = os.ReadDir
	plutilFn = func(path string) ([]byte, error) {
		return []byte(`{"Label":"com.trustedge.agent","ProgramArguments":["/bin/echo","hi"]}`), nil
	}

	got, err := collectLaunchPlists([]launchScope{{
		Hive:      "user",
		KeyPath:   "Library/LaunchAgents",
		Dir:       dir,
		EventType: constants.TypeRegistryPersist,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got=%+v", got)
	}
	if got[0].Type != constants.TypeRegistryPersist {
		t.Fatalf("type=%s", got[0].Type)
	}
}
