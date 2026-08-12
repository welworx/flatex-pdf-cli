package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// hitLog records request paths hit on the mock server, safe for concurrent
// handler goroutines.
type hitLog struct {
	mu    sync.Mutex
	paths []string
}

func (h *hitLog) add(p string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.paths = append(h.paths, p)
}

func (h *hitLog) hasPrefix(prefix string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, p := range h.paths {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}
	return false
}

// newUpgradeTestServer serves a fake releases/latest response for tagName,
// with a matching platform asset (content binContent) and SHA256SUMS.txt.
// If corruptSum is true, the published checksum doesn't match binContent.
func newUpgradeTestServer(t *testing.T, tagName string, binContent []byte, corruptSum bool) (*httptest.Server, *hitLog) {
	t.Helper()
	hits := &hitLog{}
	assetName := platformAssetName()

	sum := sha256.Sum256(binContent)
	sumHex := hex.EncodeToString(sum[:])
	if corruptSum {
		sumHex = strings.Repeat("0", 64)
	}
	sumsContent := fmt.Sprintf("%s  %s\n", sumHex, assetName)

	mux := http.NewServeMux()
	var srv *httptest.Server
	mux.HandleFunc("/repos/welworx/flatex-pdf-cli/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		hits.add(r.URL.Path)
		rel := ghRelease{
			TagName: tagName,
			Assets: []ghAsset{
				{Name: assetName, BrowserDownloadURL: srv.URL + "/download/" + assetName},
				{Name: "SHA256SUMS.txt", BrowserDownloadURL: srv.URL + "/download/SHA256SUMS.txt"},
			},
		}
		_ = json.NewEncoder(w).Encode(rel)
	})
	mux.HandleFunc("/download/"+assetName, func(w http.ResponseWriter, r *http.Request) {
		hits.add(r.URL.Path)
		_, _ = w.Write(binContent)
	})
	mux.HandleFunc("/download/SHA256SUMS.txt", func(w http.ResponseWriter, r *http.Request) {
		hits.add(r.URL.Path)
		_, _ = w.Write([]byte(sumsContent))
	})
	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, hits
}

// newUpgradeTestServerNoMatchingAsset serves a release whose only asset
// name can never match platformAssetName(), simulating an unsupported
// platform.
func newUpgradeTestServerNoMatchingAsset(t *testing.T, tagName string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/welworx/flatex-pdf-cli/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		rel := ghRelease{
			TagName: tagName,
			Assets: []ghAsset{
				{Name: "flatex-pdf-cli_plan9_386", BrowserDownloadURL: "http://unused.invalid/asset"},
				{Name: "SHA256SUMS.txt", BrowserDownloadURL: "http://unused.invalid/sums"},
			},
		}
		_ = json.NewEncoder(w).Encode(rel)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func useTestGithubAPI(t *testing.T, url string) {
	t.Helper()
	orig := githubAPIBase
	githubAPIBase = url
	t.Cleanup(func() { githubAPIBase = orig })
}

func redirectStdin(t *testing.T, input string) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.WriteString(input); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()
	orig := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = orig })
}

// captureStd redirects os.Stdout/os.Stderr for the duration of fn and
// returns what was written to each.
func captureStd(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	origOut, origErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = outW, errW

	fn()

	os.Stdout, os.Stderr = origOut, origErr
	_ = outW.Close()
	_ = errW.Close()
	var bufOut, bufErr bytes.Buffer
	_, _ = bufOut.ReadFrom(outR)
	_, _ = bufErr.ReadFrom(errR)
	return bufOut.String(), bufErr.String()
}

func writeScratchExecutable(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	target := filepath.Join(dir, "flatex-pdf-cli")
	if err := os.WriteFile(target, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	return target
}

func assertNoLeftoverFiles(t *testing.T, target string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Dir(target))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(target) {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Fatalf("leftover files in target dir: %v", names)
	}
}

func TestDoUpgradeNewerVersionConfirmed(t *testing.T) {
	binContent := []byte("new binary content")
	srv, hits := newUpgradeTestServer(t, "v0.2.0", binContent, false)
	useTestGithubAPI(t, srv.URL)
	target := writeScratchExecutable(t, "old binary content")
	redirectStdin(t, "y\n")

	var got int
	stdout, stderr := captureStd(t, func() {
		got = doUpgrade(false, false, target, "v0.1.0")
	})
	if got != 0 {
		t.Fatalf("doUpgrade = %d, want 0; stdout=%q stderr=%q", got, stdout, stderr)
	}
	if !strings.Contains(stderr, "Upgrade to v0.2.0? [y/N]") {
		t.Fatalf("prompt not shown: stderr=%q", stderr)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(binContent) {
		t.Fatalf("target content = %q, want %q", data, binContent)
	}
	assertNoLeftoverFiles(t, target)
	if !hits.hasPrefix("/download/") {
		t.Fatal("expected download requests to have been made")
	}
}

func TestRenameAsideRollback(t *testing.T) {
	target := writeScratchExecutable(t, "original content")
	dir := filepath.Dir(target)

	tmp, err := writeTempBinary(dir, []byte("new content"))
	if err != nil {
		t.Fatal(err)
	}
	// Force the second rename (tmp -> target) to fail.
	if err := os.Remove(tmp); err != nil {
		t.Fatal(err)
	}

	if err := renameAside(tmp, target); err == nil {
		t.Fatal("renameAside: expected error")
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "original content" {
		t.Fatalf("target = %q, want restored original", data)
	}
	if _, err := os.Stat(target + ".old"); !os.IsNotExist(err) {
		t.Fatalf(".old file left behind after rollback")
	}
}

func TestDoUpgradeSameVersionUpToDate(t *testing.T) {
	srv, hits := newUpgradeTestServer(t, "v0.2.0", []byte("irrelevant"), false)
	useTestGithubAPI(t, srv.URL)
	target := writeScratchExecutable(t, "unchanged")

	stdout, _ := captureStd(t, func() {
		if got := doUpgrade(false, false, target, "v0.2.0"); got != 0 {
			t.Fatalf("doUpgrade = %d, want 0", got)
		}
	})
	if !strings.Contains(stdout, "up to date") {
		t.Fatalf("stdout = %q, want up-to-date message", stdout)
	}
	if hits.hasPrefix("/download/") {
		t.Fatal("no download should have been attempted")
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "unchanged" {
		t.Fatal("target was modified")
	}
}

func TestDoUpgradeOlderVersionUpToDate(t *testing.T) {
	srv, hits := newUpgradeTestServer(t, "v0.1.0", []byte("irrelevant"), false)
	useTestGithubAPI(t, srv.URL)
	target := writeScratchExecutable(t, "unchanged")

	if got := doUpgrade(false, false, target, "v0.2.0"); got != 0 {
		t.Fatalf("doUpgrade = %d, want 0", got)
	}
	if hits.hasPrefix("/download/") {
		t.Fatal("no download should have been attempted")
	}
}

func TestDoUpgradeDevVersionAlwaysUpgradable(t *testing.T) {
	binContent := []byte("release binary")
	// Tag is lower than any real version dev would compare against, but
	// dev must upgrade anyway - it never parses as semver for comparison.
	srv, _ := newUpgradeTestServer(t, "v0.0.1", binContent, false)
	useTestGithubAPI(t, srv.URL)
	target := writeScratchExecutable(t, "old binary")

	// -y so no stdin read is needed here.
	if got := doUpgrade(false, true, target, "dev"); got != 0 {
		t.Fatalf("doUpgrade = %d, want 0", got)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(binContent) {
		t.Fatalf("target content = %q, want %q", data, binContent)
	}
}

func TestDoUpgradeChecksumMismatchAborts(t *testing.T) {
	srv, _ := newUpgradeTestServer(t, "v0.2.0", []byte("new binary"), true /* corrupt sum */)
	useTestGithubAPI(t, srv.URL)
	target := writeScratchExecutable(t, "old binary")
	redirectStdin(t, "y\n")

	got := doUpgrade(false, false, target, "v0.1.0")
	if got == 0 {
		t.Fatal("doUpgrade = 0, want non-zero on checksum mismatch")
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "old binary" {
		t.Fatalf("target was modified despite checksum mismatch: %q", data)
	}
	assertNoLeftoverFiles(t, target)
}

func TestDoUpgradeCheckReportsAvailability(t *testing.T) {
	srv, hits := newUpgradeTestServer(t, "v0.2.0", []byte("irrelevant"), false)
	useTestGithubAPI(t, srv.URL)
	target := writeScratchExecutable(t, "unchanged")

	stdout, _ := captureStd(t, func() {
		if got := doUpgrade(true, false, target, "v0.1.0"); got != 1 {
			t.Fatalf("doUpgrade(-check) = %d, want 1", got)
		}
	})
	if !strings.Contains(stdout, "v0.2.0") {
		t.Fatalf("stdout = %q, want it to mention the available version", stdout)
	}
	if hits.hasPrefix("/download/") {
		t.Fatal("-check must not download anything")
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "unchanged" {
		t.Fatal("-check must not touch the target")
	}
}

func TestDoUpgradeCheckWinsOverYes(t *testing.T) {
	srv, hits := newUpgradeTestServer(t, "v0.2.0", []byte("irrelevant"), false)
	useTestGithubAPI(t, srv.URL)
	target := writeScratchExecutable(t, "unchanged")

	// -check and -y together: -check wins, -y is ignored, no download.
	if got := doUpgrade(true, true, target, "v0.1.0"); got != 1 {
		t.Fatalf("doUpgrade(-check -y) = %d, want 1", got)
	}
	if hits.hasPrefix("/download/") {
		t.Fatal("-check must not download anything, even with -y")
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "unchanged" {
		t.Fatal("-check must not touch the target")
	}
}

func TestDoUpgradeCheckErrorExitCode(t *testing.T) {
	// No server override - githubAPIBase points nowhere reachable in test
	// isolation, but to keep this hermetic point it at a server that 404s.
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	useTestGithubAPI(t, srv.URL)
	target := writeScratchExecutable(t, "unchanged")

	if got := doUpgrade(true, false, target, "v0.1.0"); got != 2 {
		t.Fatalf("doUpgrade(-check) on error = %d, want 2", got)
	}
}

func TestDoUpgradeMissingAssetForPlatform(t *testing.T) {
	srv := newUpgradeTestServerNoMatchingAsset(t, "v0.2.0")
	useTestGithubAPI(t, srv.URL)
	target := writeScratchExecutable(t, "unchanged")

	got := doUpgrade(false, false, target, "v0.1.0")
	if got == 0 {
		t.Fatal("doUpgrade = 0, want non-zero for missing platform asset")
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "unchanged" {
		t.Fatal("target was modified despite missing asset")
	}
}

func TestDoUpgradeYesSkipsPrompt(t *testing.T) {
	binContent := []byte("new binary")
	srv, _ := newUpgradeTestServer(t, "v0.2.0", binContent, false)
	useTestGithubAPI(t, srv.URL)
	target := writeScratchExecutable(t, "old binary")

	// stdin is an empty, already-closed pipe: any attempt to read it (i.e.
	// -y not actually skipping the prompt) yields EOF, which doUpgrade
	// treats as an error - so a wrongly-shown prompt fails this test
	// instead of silently passing.
	redirectStdin(t, "")

	got := doUpgrade(false, true, target, "v0.1.0")
	if got != 0 {
		t.Fatalf("doUpgrade(-y) = %d, want 0 (prompt should have been skipped)", got)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(binContent) {
		t.Fatal("target not updated")
	}
}

func TestVersionNewer(t *testing.T) {
	cases := []struct {
		current, latest string
		want            bool
	}{
		{"v0.1.0", "v0.2.0", true},
		{"v0.2.0", "v0.1.0", false},
		{"v0.2.0", "v0.2.0", false},
		{"v0.2.0", "v0.2.1", true},
		{"v0.2.1", "v0.2.0", false},
		{"v1.0.0", "v0.99.99", false},
		// A build from a source checkout carries a `git describe` suffix.
		// It must compare as its base tag rather than failing to parse.
		{"v0.4.2-7-gb62eea4", "v0.4.2", false},
		{"v0.4.2-7-gb62eea4", "v0.4.3", true},
		{"v0.4.2+build.5", "v0.4.2", false},
		{"v0.4.2-rc.1", "v0.5.0", true},
	}
	for _, c := range cases {
		got, err := versionNewer(c.current, c.latest)
		if err != nil {
			t.Fatalf("versionNewer(%q, %q): %v", c.current, c.latest, err)
		}
		if got != c.want {
			t.Errorf("versionNewer(%q, %q) = %v, want %v", c.current, c.latest, got, c.want)
		}
	}
}

// A version that is genuinely malformed must still be rejected, and the error
// must quote what the user actually passed rather than a truncated fragment.
func TestParseSemverRejectsMalformed(t *testing.T) {
	for _, bad := range []string{"", "v1", "v1.2", "v1.2.x", "vx.2.3", "1.2.3.4.5-a"} {
		if _, _, _, err := parseSemver(bad); err == nil {
			t.Errorf("parseSemver(%q): expected an error, got nil", bad)
		}
	}

	_, _, _, err := parseSemver("v1.2.x")
	if err == nil || !strings.Contains(err.Error(), "v1.2.x") {
		t.Errorf("expected the error to quote the original input, got: %v", err)
	}
}

func TestVerifyChecksum(t *testing.T) {
	data := []byte("hello world")
	sum := sha256.Sum256(data)
	sumHex := hex.EncodeToString(sum[:])

	sums := []byte(sumHex + "  flatex-pdf-cli_linux_amd64\nsomeotherhash  someotherfile\n")
	if err := verifyChecksum(data, sums, "flatex-pdf-cli_linux_amd64"); err != nil {
		t.Fatalf("verifyChecksum: %v", err)
	}

	badSums := []byte(strings.Repeat("0", 64) + "  flatex-pdf-cli_linux_amd64\n")
	if err := verifyChecksum(data, badSums, "flatex-pdf-cli_linux_amd64"); err == nil {
		t.Fatal("verifyChecksum: expected mismatch error")
	}

	if err := verifyChecksum(data, sums, "flatex-pdf-cli_darwin_arm64"); err == nil {
		t.Fatal("verifyChecksum: expected not-listed error")
	}
}

func TestVersionNewerPropagatesParseErrors(t *testing.T) {
	if _, err := versionNewer("nonsense", "v1.0.0"); err == nil {
		t.Error("expected an error for a malformed current version")
	}
	if _, err := versionNewer("v1.0.0", "nonsense"); err == nil {
		t.Error("expected an error for a malformed latest version")
	}
}

func TestRunUpgradeUnknownFlag(t *testing.T) {
	if got := runUpgrade([]string{"-bogus"}); got != 2 {
		t.Fatalf("runUpgrade(-bogus) = %d, want 2", got)
	}
}

// setVersion swaps the build-injected version global for the duration of a
// test, so runUpgrade's "dev build" and "tagged release" paths can both be
// exercised.
func setVersion(t *testing.T, v string) {
	t.Helper()
	orig := version
	version = v
	t.Cleanup(func() { version = orig })
}

// An unset version means a source build, which is always treated as
// upgradable regardless of what the latest tag is.
func TestRunUpgradeCheckOnDevBuildReportsAvailable(t *testing.T) {
	srv, _ := newUpgradeTestServer(t, "v9.9.9", []byte("new binary"), false)
	useTestGithubAPI(t, srv.URL)
	setVersion(t, "")

	var got int
	stdout, stderr := captureStd(t, func() { got = runUpgrade([]string{"-check"}) })
	if got != 1 {
		t.Fatalf("runUpgrade(-check) = %d, want 1 (upgrade available); stderr=%q", got, stderr)
	}
	if !strings.Contains(stdout, "dev -> v9.9.9") {
		t.Fatalf("stdout = %q, want a dev -> v9.9.9 report", stdout)
	}
}

// Without -check, runUpgrade has to resolve the running binary's own path
// before it can install over it. Serving an *older* tag than the current
// version exercises that resolution and then stops at "up to date", so the
// test binary is never actually written to.
func TestRunUpgradeResolvesRunningBinaryWhenNotChecking(t *testing.T) {
	srv, hits := newUpgradeTestServer(t, "v0.1.0", []byte("new binary"), false)
	useTestGithubAPI(t, srv.URL)
	setVersion(t, "v9.9.9")

	var got int
	stdout, stderr := captureStd(t, func() { got = runUpgrade([]string{"-y"}) })
	if got != 0 {
		t.Fatalf("runUpgrade(-y) = %d, want 0; stdout=%q stderr=%q", got, stdout, stderr)
	}
	if !strings.Contains(stdout, "up to date (v9.9.9)") {
		t.Fatalf("stdout = %q, want an up-to-date report", stdout)
	}
	if hits.hasPrefix("/download/") {
		t.Fatal("an up-to-date check must not download anything")
	}
}

// Answering anything other than "y" leaves the installed binary alone, and
// must do so before any bytes are fetched.
func TestDoUpgradeDeclinedAtPrompt(t *testing.T) {
	srv, hits := newUpgradeTestServer(t, "v0.2.0", []byte("new binary"), false)
	useTestGithubAPI(t, srv.URL)
	target := writeScratchExecutable(t, "old binary")
	redirectStdin(t, "n\n")

	var got int
	stdout, stderr := captureStd(t, func() { got = doUpgrade(false, false, target, "v0.1.0") })
	if got != 0 {
		t.Fatalf("declined upgrade = %d, want 0; stderr=%q", got, stderr)
	}
	if !strings.Contains(stdout, "aborted") {
		t.Fatalf("stdout = %q, want %q", stdout, "aborted")
	}
	if hits.hasPrefix("/download/") {
		t.Fatal("declining the prompt must not download anything")
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "old binary" {
		t.Fatal("target modified after the upgrade was declined")
	}
}

// A prompt that can't be read (stdin closed, e.g. a non-interactive shell
// without -y) is an error, not an implicit yes.
func TestDoUpgradeUnreadablePromptIsAnError(t *testing.T) {
	srv, _ := newUpgradeTestServer(t, "v0.2.0", []byte("new binary"), false)
	useTestGithubAPI(t, srv.URL)
	target := writeScratchExecutable(t, "old binary")
	redirectStdin(t, "")

	var got int
	_, stderr := captureStd(t, func() { got = doUpgrade(false, false, target, "v0.1.0") })
	if got != 1 {
		t.Fatalf("doUpgrade with unreadable stdin = %d, want 1; stderr=%q", got, stderr)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "old binary" {
		t.Fatal("target modified despite an unreadable prompt")
	}
}

// Without SHA256SUMS.txt there is nothing to verify the download against, so
// the upgrade must refuse rather than install unverified bytes.
func TestDoUpgradeReleaseWithoutChecksumsFile(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/welworx/flatex-pdf-cli/releases/latest", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(ghRelease{
			TagName: "v0.2.0",
			Assets:  []ghAsset{{Name: platformAssetName(), BrowserDownloadURL: "http://unused.invalid/asset"}},
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	useTestGithubAPI(t, srv.URL)
	target := writeScratchExecutable(t, "unchanged")

	var got int
	_, stderr := captureStd(t, func() { got = doUpgrade(false, true, target, "v0.1.0") })
	if got != 1 {
		t.Fatalf("doUpgrade = %d, want 1", got)
	}
	if !strings.Contains(stderr, "SHA256SUMS.txt") {
		t.Fatalf("stderr = %q, want it to name the missing SHA256SUMS.txt", stderr)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "unchanged" {
		t.Fatal("target modified despite the missing checksums file")
	}
}

func TestDoUpgradeMalformedCurrentVersionIsAnError(t *testing.T) {
	srv, _ := newUpgradeTestServer(t, "v0.2.0", []byte("new binary"), false)
	useTestGithubAPI(t, srv.URL)
	target := writeScratchExecutable(t, "unchanged")

	var got int
	_, stderr := captureStd(t, func() { got = doUpgrade(false, true, target, "not-a-version") })
	if got != 1 {
		t.Fatalf("doUpgrade = %d, want 1", got)
	}
	if !strings.Contains(stderr, "not-a-version") {
		t.Fatalf("stderr = %q, want it to quote the unparseable version", stderr)
	}
}

// newUpgradeTestServerBrokenDownloads serves a well-formed release whose
// asset URLs 404. With binOK the platform binary downloads fine and only the
// checksums file is broken, so each download call site can be failed
// independently.
func newUpgradeTestServerBrokenDownloads(t *testing.T, binOK bool) *httptest.Server {
	t.Helper()
	assetName := platformAssetName()
	mux := http.NewServeMux()
	var srv *httptest.Server
	mux.HandleFunc("/repos/welworx/flatex-pdf-cli/releases/latest", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(ghRelease{
			TagName: "v0.2.0",
			Assets: []ghAsset{
				{Name: assetName, BrowserDownloadURL: srv.URL + "/download/" + assetName},
				{Name: "SHA256SUMS.txt", BrowserDownloadURL: srv.URL + "/download/SHA256SUMS.txt"},
			},
		})
	})
	if binOK {
		mux.HandleFunc("/download/"+assetName, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("new binary"))
		})
	}
	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestDoUpgradeDownloadFailures(t *testing.T) {
	for _, c := range []struct {
		name  string
		binOK bool
	}{
		{"binary download fails", false},
		{"checksums download fails", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			srv := newUpgradeTestServerBrokenDownloads(t, c.binOK)
			useTestGithubAPI(t, srv.URL)
			target := writeScratchExecutable(t, "unchanged")

			var got int
			_, stderr := captureStd(t, func() { got = doUpgrade(false, true, target, "v0.1.0") })
			if got != 1 {
				t.Fatalf("doUpgrade = %d, want 1; stderr=%q", got, stderr)
			}
			data, err := os.ReadFile(target)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != "unchanged" {
				t.Fatal("target modified despite a failed download")
			}
			assertNoLeftoverFiles(t, target)
		})
	}
}

// A verified download that can't be installed (target directory gone) must
// surface the error rather than report success.
func TestDoUpgradeUninstallableTargetIsAnError(t *testing.T) {
	srv, _ := newUpgradeTestServer(t, "v0.2.0", []byte("new binary"), false)
	useTestGithubAPI(t, srv.URL)
	target := filepath.Join(t.TempDir(), "no-such-dir", "flatex-pdf-cli")

	var got int
	stdout, stderr := captureStd(t, func() { got = doUpgrade(false, true, target, "v0.1.0") })
	if got != 1 {
		t.Fatalf("doUpgrade = %d, want 1", got)
	}
	if stderr == "" {
		t.Fatal("expected the install failure on stderr")
	}
	if strings.Contains(stdout, "upgraded to") {
		t.Fatalf("reported success despite failing to install: %q", stdout)
	}
}

func TestFetchLatestReleaseErrors(t *testing.T) {
	t.Run("unreachable host", func(t *testing.T) {
		useTestGithubAPI(t, "http://127.0.0.1:0")
		if _, err := fetchLatestRelease(); err == nil {
			t.Fatal("expected a transport error")
		}
	})

	t.Run("malformed json", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/welworx/flatex-pdf-cli/releases/latest", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("{not json"))
		})
		srv := httptest.NewServer(mux)
		t.Cleanup(srv.Close)
		useTestGithubAPI(t, srv.URL)

		if _, err := fetchLatestRelease(); err == nil {
			t.Fatal("expected a decode error")
		}
	})
}

func TestDownloadBytesErrors(t *testing.T) {
	if _, err := downloadBytes("http://127.0.0.1:0/asset"); err == nil {
		t.Fatal("expected a transport error")
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	if _, err := downloadBytes(srv.URL + "/asset"); err == nil {
		t.Fatal("expected a non-200 status to be an error")
	}
}

func TestWriteTempBinaryRejectsMissingDir(t *testing.T) {
	if _, err := writeTempBinary(filepath.Join(t.TempDir(), "no-such-dir"), []byte("x")); err == nil {
		t.Fatal("expected an error for a nonexistent directory")
	}
}
