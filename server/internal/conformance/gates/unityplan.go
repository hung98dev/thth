package gates

import "strings"

// Owner tasks whose DONE status activates the Unity Q3 gates (audit_gates.md
// § Gate Activation). The verifier's gate table and the Unity mode plan read
// the same constants so the workflow never launches a mode whose gate would
// report SKIP(owner-not-done), and never skips a mode whose gate is active.
const (
	UnityEditModeOwner = "IMP-000"
	UnityPlayModeOwner = "IMP-065"
)

// UnityPlan lists the Unity test modes the CI Unity job launches (ADR-0074).
type UnityPlan struct {
	EditMode bool
	PlayMode bool
}

// PlanUnityModes returns the modes whose Q3 gate is active for the head:
// the owner task is DONE on main or in the head, and the PR is not
// status-only. The Unity scope skip (ADR-0073) is decided separately by the
// workflow and re-checked by the verifier.
func PlanUnityModes(required func(string) bool, statusOnly bool) UnityPlan {
	if statusOnly {
		return UnityPlan{}
	}
	return UnityPlan{
		EditMode: required(UnityEditModeOwner),
		PlayMode: required(UnityPlayModeOwner),
	}
}

// Modes returns the planned modes in launch order, e.g. "EditMode PlayMode".
func (p UnityPlan) Modes() string {
	var m []string
	if p.EditMode {
		m = append(m, "EditMode")
	}
	if p.PlayMode {
		m = append(m, "PlayMode")
	}
	return strings.Join(m, " ")
}

// GithubOutput renders the plan as GITHUB_OUTPUT lines.
func (p UnityPlan) GithubOutput() string {
	b := func(v bool) string {
		if v {
			return "true"
		}
		return "false"
	}
	return "editmode=" + b(p.EditMode) + "\nplaymode=" + b(p.PlayMode) + "\nmodes=" + p.Modes() + "\n"
}
