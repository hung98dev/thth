// Package gates hosts the executable repository gates Q0-Q6
// (docs/10_implementation/audit_gates.md) plus the shared check types used by
// the verifier entrypoint in server/cmd/verify.
package gates

// Status is the outcome of one executable check.
type Status string

const (
	StatusPass     Status = "PASS"
	StatusFail     Status = "FAIL"
	StatusSkip     Status = "SKIP"
	StatusDeferred Status = "DEFERRED"
)

// Check is one named gate assertion.
type Check struct {
	ID     string `json:"id"`
	Status Status `json:"status"`
	Detail string `json:"detail,omitempty"`
	// Owner names the task that owns completing this gate when the check is a
	// named skip (audit_gates.md — allowed skips must be named).
	Owner string `json:"owner,omitempty"`
}

// Gate groups checks under one Q gate.
type Gate struct {
	ID     string  `json:"id"`
	Title  string  `json:"title"`
	Checks []Check `json:"checks"`
}

// Status returns PASS/FAIL/SKIP/DEFERRED for a whole gate: any FAIL fails the
// gate; a gate with only skips is SKIP; all-deferred is DEFERRED.
func (g Gate) Status() Status {
	anyFail, anyPass, anyDeferred := false, false, false
	for _, c := range g.Checks {
		switch c.Status {
		case StatusFail:
			anyFail = true
		case StatusPass:
			anyPass = true
		case StatusDeferred:
			anyDeferred = true
		}
	}
	switch {
	case anyFail:
		return StatusFail
	case anyPass:
		return StatusPass
	case anyDeferred:
		return StatusDeferred
	default:
		return StatusSkip
	}
}

func Pass(id, detail string) Check {
	return Check{ID: id, Status: StatusPass, Detail: detail}
}

func Fail(id, detail string) Check {
	return Check{ID: id, Status: StatusFail, Detail: detail}
}

// Skip reports a named skip: owner + reason are mandatory (audit_gates.md —
// a skip is legal only when named).
func Skip(id, owner, reason string) Check {
	return Check{ID: id, Status: StatusSkip, Owner: owner, Detail: reason}
}

// SkipOwnerNotDone is the canonical activation skip (audit_gates.md § Gate
// Activation): the check's owner task is not DONE on main or in the PR head.
func SkipOwnerNotDone(id, owner string) Check {
	return Skip(id, owner, "owner-not-done "+owner)
}

// SkipStatusOnly reports SKIP(status-only): the PR only touches claim/
// blocker control files, so non-Q0 gates do not run (audit_gates.md §
// Protected Paths).
func SkipStatusOnly(id string) Check {
	return Check{ID: id, Status: StatusSkip, Detail: "status-only"}
}

// Deferred reports DEFERRED(local-missing): a canonical tool is absent on a
// local run under verify.ps1 -LocalDeferMissing. Never valid in CI.
func Deferred(id, what string) Check {
	return Check{ID: id, Status: StatusDeferred, Detail: "local-missing: " + what}
}
