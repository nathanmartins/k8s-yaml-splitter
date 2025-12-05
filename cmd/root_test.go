package cmd

import (
	"bytes"
	"io"
	"log/slog"
	"os"
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
