package gates

import "testing"

// ADR-0074: the Unity job launches exactly the modes whose Q3 gate is active.
func TestPlanUnityModes(t *testing.T) {
	done := func(ids ...string) func(string) bool {
		return func(o string) bool {
			for _, id := range ids {
				if id == o {
					return true
				}
			}
			return false
		}
	}
	cases := []struct {
		name       string
		required   func(string) bool
		statusOnly bool
		want       UnityPlan
		output     string
	}{
		{"no owner done", done(), false, UnityPlan{}, "editmode=false\nplaymode=false\nmodes=\n"},
		{"editmode owner done", done(UnityEditModeOwner), false, UnityPlan{EditMode: true}, "editmode=true\nplaymode=false\nmodes=EditMode\n"},
		{"both owners done", done(UnityEditModeOwner, UnityPlayModeOwner), false, UnityPlan{EditMode: true, PlayMode: true}, "editmode=true\nplaymode=true\nmodes=EditMode PlayMode\n"},
		{"status-only PR", done(UnityEditModeOwner, UnityPlayModeOwner), true, UnityPlan{}, "editmode=false\nplaymode=false\nmodes=\n"},
	}
	for _, c := range cases {
		got := PlanUnityModes(c.required, c.statusOnly)
		if got != c.want {
			t.Errorf("%s: plan = %+v, want %+v", c.name, got, c.want)
		}
		if out := got.GithubOutput(); out != c.output {
			t.Errorf("%s: output = %q, want %q", c.name, out, c.output)
		}
	}
}
