package gates

import (
	"regexp"
	"strings"
)

// UnityRelevantPattern matches the repo paths whose change requires the Unity
// steps and gates to run on a pull request (ADR-0073). verify.yml's
// "Unity scope" step carries this exact literal on both jobs; a conformance
// test keeps them equal.
const UnityRelevantPattern = `^(client/|proto/|scripts/codegen\.|scripts/verify\.ps1$|\.github/workflows/|server/cmd/verify/|server/internal/conformance/|docs/00_context/technology_versions\.md$)`

var unityRelevantRe = regexp.MustCompile(UnityRelevantPattern)

// Values of THINHTHAN_UNITY_SCOPE exported by the workflow "Unity scope" step.
const (
	UnityScopeFull           = "full"
	UnityScopeNoClientChange = "no-client-change"
)

// UnityRelevant reports whether any changed path requires Unity.
func UnityRelevant(files []string) bool {
	for _, f := range files {
		if unityRelevantRe.MatchString(f) {
			return true
		}
	}
	return false
}

// UnityAlwaysFullBranch reports head branches that must always run Unity in
// full: the IMP-068 cutover PRs and every two-phase `-done` status PR.
func UnityAlwaysFullBranch(branch string) bool {
	return strings.HasPrefix(branch, "imp/IMP-068-") || strings.HasSuffix(branch, "-done")
}

// ResolveUnityScope validates the workflow-declared scope against the event
// and the diff the verifier computes itself. skip is true only for a
// legitimate SKIP(no-client-change); problem is non-empty when the workflow
// skipped Unity where it must not (the Unity gates then FAIL).
func ResolveUnityScope(e *Env, declared string, files []string, filesErr error) (skip bool, problem string) {
	if declared != UnityScopeNoClientChange {
		return false, ""
	}
	switch {
	case !e.IsPR():
		return false, "no-client-change is only valid on pull_request events (event=" + e.EventName + ")"
	case UnityAlwaysFullBranch(e.HeadRef):
		return false, "branch " + e.HeadRef + " must run Unity in full"
	case filesErr != nil:
		return false, "cannot verify Unity scope: " + filesErr.Error()
	case UnityRelevant(files):
		return false, "Unity skipped but the diff touches Unity-relevant paths"
	}
	return true, ""
}

// SkipNoClientChange reports SKIP(no-client-change): the PR diff touches no
// Unity-relevant path, so the Unity steps did not run (ADR-0073).
func SkipNoClientChange(id string) Check {
	return Check{ID: id, Status: StatusSkip, Detail: UnityScopeNoClientChange}
}
