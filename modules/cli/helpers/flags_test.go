package helpers

import (
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/kuetix/engine/engine/domain"
)

// withArgs runs fn with os.Args set to append(["kue"], args...), restoring
// the original afterward.
func withArgs(t *testing.T, args []string, fn func()) {
	t.Helper()
	orig := os.Args
	os.Args = append([]string{"kue"}, args...)
	defer func() { os.Args = orig }()
	fn()
}

// TestGetArgsBoolFlagBeforePositional guards against a regression where a
// boolean flag placed before its command's positional argument swallowed
// that argument as if it were the flag's own value — e.g.
// `kue workflow upload --public ai/agent` reported "workflow name is
// required" because "ai/agent" landed in options, never in args.
func TestGetArgsBoolFlagBeforePositional(t *testing.T) {
	boolFlags := map[string]bool{"public": true, "P": true}
	withArgs(t, []string{"workflow", "upload", "--public", "ai/agent"}, func() {
		_, command, args, options := GetArgs(boolFlags)
		if command != "workflow.upload" {
			t.Fatalf("command = %q", command)
		}
		if len(args) != 1 || args[0] != "ai/agent" {
			t.Fatalf("args = %v, want [ai/agent]", args)
		}
		if len(options) != 1 || options[0] != "--public" {
			t.Fatalf("options = %v, want [--public]", options)
		}
	})
}

// TestGetArgsGlobalBoolFlagBeforeCommand guards against a regression where a
// global boolean flag (-v/--debug/-q/-h) placed before the command name
// swallowed the command word itself as its value — e.g. `kue -v workflow
// status` resolved to the bogus command "status" ("workflow" having been
// consumed as -v's "value"), instead of "workflow.status".
func TestGetArgsGlobalBoolFlagBeforeCommand(t *testing.T) {
	boolFlags := map[string]bool{"v": true, "verbose": true}
	withArgs(t, []string{"-v", "workflow", "status"}, func() {
		_, command, args, options := GetArgs(boolFlags)
		if command != "workflow.status" {
			t.Fatalf("command = %q, want workflow.status", command)
		}
		if len(args) != 0 {
			t.Fatalf("args = %v, want none", args)
		}
		if len(options) != 1 || options[0] != "-v" {
			t.Fatalf("options = %v, want [-v]", options)
		}
	})
}

// TestGetArgsUnknownFlagStillConsumesValue preserves the long-standing
// behavior for flags GetArgs can't classify as boolean (host, output, ...):
// the token right after them is still treated as their value, not as a
// second positional argument.
func TestGetArgsUnknownFlagStillConsumesValue(t *testing.T) {
	withArgs(t, []string{"install", "acme/thing", "--output", "/tmp/proj"}, func() {
		_, command, args, options := GetArgs(nil)
		if command != "install.acme/thing" {
			t.Fatalf("command = %q", command)
		}
		if len(args) != 0 {
			t.Fatalf("args = %v, want none (both consumed as command+option value)", args)
		}
		if strings.Join(options, ",") != "--output,/tmp/proj" {
			t.Fatalf("options = %v", options)
		}
	})
}

// TestGetArgsFlagWithEqualsIsSelfContained covers --flag=value: the
// following token must not be swallowed as a second value for it.
func TestGetArgsFlagWithEqualsIsSelfContained(t *testing.T) {
	withArgs(t, []string{"run", "acme/thing", "--host=http://x", "--check"}, func() {
		boolFlags := map[string]bool{"check": true}
		_, _, args, options := GetArgs(boolFlags)
		if len(args) != 0 {
			t.Fatalf("args = %v", args)
		}
		if strings.Join(options, ",") != "--host=http://x,--check" {
			t.Fatalf("options = %v", options)
		}
	})
}

// RenderHelp is what command transitions call for `--help`. It must tolerate a
// nil session (unit tests call transitions directly, without an engine) and a
// missing config["flagSet"] — regression guard for the panic that
// `kue show --help` used to hit ("interface conversion: interface {} is nil,
// not *flag.FlagSet").
func TestRenderHelpNilSession(t *testing.T) {
	t.Run("literal usage passes through", func(t *testing.T) {
		got := RenderHelp(nil, map[string]interface{}{"usage": "USAGE: kue show"}, nil)
		if !strings.Contains(got, "USAGE: kue show") {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("missing flagSet does not panic", func(t *testing.T) {
		_ = RenderHelp(nil, map[string]interface{}{"usage": "x"}, map[string]interface{}{})
	})

	t.Run("present flagSet does not panic", func(t *testing.T) {
		fs := flag.NewFlagSet("show", flag.ContinueOnError)
		_ = RenderHelp(nil, map[string]interface{}{"usage": "x", "flagSet": fs}, nil)
	})

	t.Run("nil config does not panic", func(t *testing.T) {
		_ = RenderHelp(nil, nil, nil)
	})

	t.Run("wrong flagSet type does not panic", func(t *testing.T) {
		_ = RenderHelp(nil, map[string]interface{}{"usage": "x", "flagSet": "not a flagset"}, nil)
	})
}

func TestGetUsageLiteralVsFile(t *testing.T) {
	// A literal (non file://) usage string is returned as-is.
	got := GetUsage(domain.Application{}, "plain text", nil, "workflows")
	if !strings.Contains(got, "plain text") {
		t.Fatalf("literal usage: %q", got)
	}

	// An unreadable file:// reference degrades to the raw string rather than
	// panicking or returning an "err:" sentinel.
	got = GetUsage(domain.Application{}, "file://does/not/exist.txt", nil, "workflows")
	if strings.HasPrefix(got, "err:") {
		t.Fatalf("should not leak err sentinel: %q", got)
	}
}
