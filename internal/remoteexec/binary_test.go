package remoteexec

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"crypto/ed25519"
	"crypto/rand"

	"golang.org/x/crypto/ssh"
)

// DirBinaryProvider serves the per-arch binary matching the request and refuses a
// request for an arch it does not hold — the arch check's refusal side.
func TestDirBinaryProviderSelectsByArch(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "prober-linux-amd64"), "AMD64")
	writeFile(t, filepath.Join(dir, "prober-linux-arm64"), "ARM64")
	p := DirBinaryProvider{Dir: dir}

	if got := readBinary(t, p, "linux", "amd64"); got != "AMD64" {
		t.Errorf("linux/amd64 binary = %q", got)
	}
	if got := readBinary(t, p, "linux", "arm64"); got != "ARM64" {
		t.Errorf("linux/arm64 binary = %q", got)
	}
	if _, err := p.Binary("linux", "riscv64"); !errors.Is(err, ErrNoBinary) {
		t.Errorf("riscv64 err = %v, want ErrNoBinary", err)
	}
}

// With no per-arch directory, the own-arch fallback is served for the instance's own
// platform only — never handed back for a different architecture.
func TestDirBinaryProviderFallbackOwnArchOnly(t *testing.T) {
	dir := t.TempDir()
	fallback := filepath.Join(dir, "prober")
	writeFile(t, fallback, "OWN")
	p := DirBinaryProvider{Fallback: fallback}

	if got := readBinary(t, p, runtime.GOOS, runtime.GOARCH); got != "OWN" {
		t.Errorf("own-arch fallback = %q, want OWN", got)
	}
	// A different arch must not be served the own-arch fallback.
	otherArch := "arm64"
	if runtime.GOARCH == "arm64" {
		otherArch = "amd64"
	}
	if _, err := p.Binary(runtime.GOOS, otherArch); !errors.Is(err, ErrNoBinary) {
		t.Errorf("cross-arch fallback err = %v, want ErrNoBinary", err)
	}
}

// Fingerprint renders the canonical SHA256 fingerprint of a pinned host key, and
// collapses to "" for an empty or unparseable pin.
func TestFingerprint(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	pinned := trimAuthorizedKey(sshPub)

	got := Fingerprint(pinned)
	want := ssh.FingerprintSHA256(sshPub)
	if got != want {
		t.Errorf("Fingerprint = %q, want %q", got, want)
	}
	if got == "" || got[:7] != "SHA256:" {
		t.Errorf("fingerprint %q is not the canonical SHA256 form", got)
	}
	if Fingerprint("") != "" {
		t.Error("empty pin should render blank")
	}
	if Fingerprint("not a key") != "" {
		t.Error("unparseable pin should render blank")
	}
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func readBinary(t *testing.T, p BinaryProvider, goos, goarch string) string {
	t.Helper()
	rc, err := p.Binary(goos, goarch)
	if err != nil {
		t.Fatalf("Binary(%s,%s): %v", goos, goarch, err)
	}
	defer rc.Close()
	b, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func trimAuthorizedKey(key ssh.PublicKey) string {
	line := ssh.MarshalAuthorizedKey(key)
	if n := len(line); n > 0 && line[n-1] == '\n' {
		line = line[:n-1]
	}
	return string(line)
}
