package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestLogger_BasicLevelsAndFields(t *testing.T) {
	var buf bytes.Buffer
	log := New(Config{Level: "debug", Format: "text", Output: &buf})

	log.Debug("dbg", String("k", "v"))
	log.Info("info", Int("n", 42))
	log.Warn("warn", Bool("ok", true))
	log.Error("err", Error(nil))

	out := buf.String()
	// Expect all levels present (debug is the lowest configured)
	for _, s := range []string{"[DEBUG] dbg k=v", "[INFO] info n=42", "[WARN] warn ok=true", "[ERROR] err error=nil"} {
		if !strings.Contains(out, s) {
			t.Fatalf("expected output to contain %q, got: %s", s, out)
		}
	}
}

func TestLogger_WithComponentPrefix(t *testing.T) {
	var buf bytes.Buffer
	base := New(Config{Level: "info", Output: &buf})
	comp := base.WithComponent("network.server")

	comp.Info("started")

	out := buf.String()
	if !strings.Contains(out, "[network.server]") {
		t.Fatalf("expected component prefix in output, got: %s", out)
	}
	if !strings.Contains(out, "[INFO] started") {
		t.Fatalf("expected info message in output, got: %s", out)
	}
}
