package transitions

import "testing"

// TestCollectBoolFlagNames checks that every "type": "bool" flag anywhere in
// a commands.wsl-shaped tree — the global "*" block and every group's
// subcommands, including one that only "$extends" a sibling — is collected
// by both its long and short spelling. GetArgs relies on this set to tell a
// self-contained boolean flag from one that still needs a following value
// token (see helpers.TestGetArgsBoolFlagBeforePositional).
func TestCollectBoolFlagNames(t *testing.T) {
	groups := map[string]interface{}{
		"*": []interface{}{
			map[string]interface{}{
				"*": map[string]interface{}{
					"options": []interface{}{
						map[string]interface{}{"long": "verbose", "short": "v", "value": false, "type": "bool"},
						map[string]interface{}{"long": "modules", "short": "mp", "value": "modules"},
					},
				},
			},
		},
		"workflow": []interface{}{
			map[string]interface{}{
				"upload": map[string]interface{}{
					"options": []interface{}{
						map[string]interface{}{"long": "public", "short": "P", "value": false, "type": "bool"},
						map[string]interface{}{"long": "host", "short": "H", "value": ""},
					},
				},
				// A sibling that only extends another entry contributes no
				// options of its own — this must not panic or lose data.
				"help": map[string]interface{}{"$extends": ""},
			},
		},
	}

	got := collectBoolFlagNames(groups)
	for _, want := range []string{"verbose", "v", "public", "P"} {
		if !got[want] {
			t.Errorf("expected %q to be collected as a bool flag, got %v", want, got)
		}
	}
	for _, notWant := range []string{"modules", "mp", "host", "H"} {
		if got[notWant] {
			t.Errorf("%q is not a bool flag, should not be collected", notWant)
		}
	}
}
