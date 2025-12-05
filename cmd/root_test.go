package cmd

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFileBytes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fp := filepath.Join(dir, "sample.yaml")
	content := "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test\n"
	if err := os.WriteFile(fp, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	b, err := ReadFileBytes(fp)
	if err != nil {
		t.Fatalf("ReadFileBytes returned error: %v", err)
	}
	if got := string(b); got != content {
		t.Fatalf("ReadFileBytes content mismatch.\nGot:\n%q\nWant:\n%q", got, content)
	}
}

func TestOverWriteToFile_TruncatesAndWrites(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fp := filepath.Join(dir, "out.yaml")

	// Prime file with longer content to ensure truncation happens.
	if err := os.WriteFile(fp, []byte(strings.Repeat("X", 1024)), 0o600); err != nil {
		t.Fatalf("prime file: %v", err)
	}

	// Use a no-op logger that writes to an in-memory buffer.
	var logBuf bytes.Buffer
	logger := newTestLogger(&logBuf)

	payload := "kind: Pod\nmetadata:\n  name: demo\n"
	OverWriteToFile(fp, payload, logger)

	got, err := os.ReadFile(fp)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(got) != payload {
		t.Fatalf("file not truncated/written correctly. got=%q want=%q", string(got), payload)
	}
}

func TestRootCommand_SplitsFromStdin(t *testing.T) {
	t.Parallel()

	// Prepare multi-document YAML with varying Kinds and names (including a colon to test sanitization).
	input := "" +
		"apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: my-app\n---\n" +
		"apiVersion: v1\nkind: Service\nmetadata:\n  name: api:edge\n---\n" +
		"apiVersion: networking.k8s.io/v1\nkind: Ingress\nmetadata:\n  name: web\n"

	outDir := t.TempDir()

	// Replace stdin with a pipe feeding our input.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	// Ensure we restore the original stdin afterward.
	origStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = origStdin
		r.Close() // ignore error in cleanup
	})

	go func() {
		defer w.Close()
		io.WriteString(w, input)
	}()

	// Run the cobra command with one argument (output directory) so it reads from stdin.
	rootCmd.SetArgs([]string{outDir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("rootCmd.Execute error: %v", err)
	}

	// Expected files (lowercased kinds and names, ':' replaced with '-').
	wantFiles := []string{
		filepath.Join(outDir, "deployment-my-app.yaml"),
		filepath.Join(outDir, "service-api-edge.yaml"),
		filepath.Join(outDir, "ingress-web.yaml"),
	}
	for _, fp := range wantFiles {
		b, err := os.ReadFile(fp)
		if err != nil {
			t.Fatalf("expected output file missing: %s: %v", fp, err)
		}
		// Basic sanity: ensure the file contains a Kind and metadata.name.
		s := string(b)
		if !strings.Contains(s, "kind:") || !strings.Contains(s, "metadata:") {
			t.Fatalf("output file %s seems invalid:\n%s", fp, s)
		}
	}
}

// newTestLogger returns a slog.Logger that writes plain text logs into dst.
// We keep it here to avoid pulling in external logging helpers in tests.
func newTestLogger(dst io.Writer) *slog.Logger {
	// Use default text handler with nil options to keep behavior similar to production logger.
	return slog.New(slog.NewTextHandler(dst, nil))
}

func TestReadFileBytes_Nonexistent(t *testing.T) {
	t.Parallel()
	_, err := ReadFileBytes(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	if err == nil {
		t.Fatalf("expected error for nonexistent file")
	}
}

func TestRootCommand_ArgsValidationErrors(t *testing.T) {
	t.Parallel()

	// Zero args -> cobra ExactArgs(1) should error from Execute.
	rootCmd.SetArgs([]string{})
	if err := rootCmd.Execute(); err == nil {
		t.Fatalf("expected error for zero args, got nil")
	}

	// Two args -> should also error (ExactArgs(1)).
	rootCmd.SetArgs([]string{"in.yaml", "out", "extra"})
	if err := rootCmd.Execute(); err == nil {
		t.Fatalf("expected error for two args, got nil")
	}
}

func TestRootCommand_EmptyStdin_NoFiles(t *testing.T) {
	t.Parallel()

	outDir := t.TempDir()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	orig := os.Stdin
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = orig
		r.Close()
	})

	// Write nothing, then close.
	go func() { _ = w.Close() }()

	rootCmd.SetArgs([]string{outDir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute with empty stdin returned error: %v", err)
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no files created, found %d", len(entries))
	}
}

// Subprocess helper to exercise code paths that call os.Exit.
// Pattern: parent test runs this test binary with -test.run=TestHelperProcess and
// GO_WANT_HELPER_PROCESS=1 plus SCENARIO to pick the branch below.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	scenario := os.Getenv("SCENARIO")

	switch scenario {
	case "INVALID_YAML":
		// Stdin provided by parent; run root command with OUT_DIR argument.
		out := os.Getenv("OUT_DIR")
		rootCmd.SetArgs([]string{out})
		// Execute will read from stdin and likely hit invalid YAML -> os.Exit(-1) in Run.
		// If for some reason it doesn't exit, force non-zero to signal failure of expectation.
		_ = rootCmd.Execute()
		os.Exit(0)
	case "UNWRITABLE_DIR":
		// OUT_PATH points to a regular file, not a directory.
		outPath := os.Getenv("OUT_PATH")
		// Feed stdin provided by parent (valid yaml). Expect mkdir to fail and exit.
		rootCmd.SetArgs([]string{outPath})
		_ = rootCmd.Execute()
		os.Exit(0)
	case "OVERWRITE_WRITE_ERROR":
		// Attempt to write to a directory path to trigger open/write error.
		dir := os.Getenv("DIR_PATH")
		logger := newTestLogger(io.Discard)
		OverWriteToFile(dir, "payload", logger)
		os.Exit(0)
	default:
		os.Exit(0)
	}
}

func TestRootCommand_InvalidYAML_ExitsNonZero(t *testing.T) {
	t.Parallel()

	outDir := t.TempDir()

	cmd := exec.Command(os.Args[0], "-test.run", "TestHelperProcess")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1", "SCENARIO=INVALID_YAML", "OUT_DIR="+outDir)
	// Malformed YAML
	cmd.Stdin = strings.NewReader("kind: : :\nmetadata: [\n")

	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected non-zero exit, got err=%v", err)
	}
	if exitErr.ExitCode() == 0 {
		t.Fatalf("expected non-zero exit code for invalid YAML")
	}
}

func TestRootCommand_UnwritableOutputDir_ExitsNonZero(t *testing.T) {
	t.Parallel()

	// Create a regular file and try to use it as the output directory.
	f, err := os.CreateTemp(t.TempDir(), "out-*")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	_ = f.Close()

	cmd := exec.Command(os.Args[0], "-test.run", "TestHelperProcess")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1", "SCENARIO=UNWRITABLE_DIR", "OUT_PATH="+f.Name())
	// Provide minimally valid single-doc YAML on stdin.
	cmd.Stdin = strings.NewReader("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: a\n")

	err = cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected non-zero exit, got err=%v", err)
	}
	if exitErr.ExitCode() == 0 {
		t.Fatalf("expected non-zero exit code for unwritable output path")
	}
}

func TestOverWriteToFile_WriteError_ExitsNonZero(t *testing.T) {
	t.Parallel()

	// Use a directory path as the "file" to trigger a write/open error inside OverWriteToFile
	dir := t.TempDir()

	cmd := exec.Command(os.Args[0], "-test.run", "TestHelperProcess")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1", "SCENARIO=OVERWRITE_WRITE_ERROR", "DIR_PATH="+dir)

	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected non-zero exit, got err=%v", err)
	}
	if exitErr.ExitCode() == 0 {
		t.Fatalf("expected non-zero exit code for write error scenario")
	}
}
