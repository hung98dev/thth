package gates

import (
	"bufio"
	"os"
	pathpkg "path"
	"path/filepath"
	"regexp"
	"strings"
)

// TaskPacket is one parsed IMP entry from docs/10_implementation/task_queue.md.
type TaskPacket struct {
	ID               string
	Status           string
	ClaimedBy        string
	Branch           string
	ClaimedAt        string
	BlockedBy        string
	Specs            []string
	ADRs             []string
	DependsOn        []string
	OwnedPaths       []string
	ForbiddenPaths   []string
	ContractInputs   []string
	ContractOutputs  []string
	ConsumersChecked []string
	EvidenceLocation string
	RequiredEvidence string // required_evidence field; "none" exempts DONE-manifest
	TestsBody        string
	ChangeBody       string
	AcceptanceBody   string
	// Raw is the packet's full text (minus the heading line) used for
	// field-level diffs between revisions.
	Raw string
}

var (
	packetHeadRe = regexp.MustCompile(`^##\s+` + "`" + `(IMP-[0-9]+)` + "`" + `(\s|$)`)
	fieldRe      = regexp.MustCompile(`^([a-z_]+):\s*(.*)$`)
	testFileRe   = regexp.MustCompile(`[A-Za-z0-9_./-]+\.(?:go|ps1|sh|cs|sql|json|md|asmdef|proto)`)
	summaryRe    = regexp.MustCompile(`^\|\s+` + "`" + `(IMP-[0-9]+)` + "`" + `\s+\|[^|]+\|\s+` + "`" + `([A-Z_]+)` + "`" + `\s+\|`)
	taskIDRefRe  = regexp.MustCompile(`\b((?:IMP|RMP)-[0-9]+)\b`)
)

var validStatus = map[string]bool{
	"NOT_STARTED": true,
	"IN_PROGRESS": true,
	"BLOCKED":     true,
	"DONE":        true,
}

// mutablePacketFields are the only packet fields a task PR may change:
// status lifecycle + claim fields (agent_execution_protocol.md §3).
var mutablePacketFields = map[string]bool{
	"status": true, "claimed_by": true, "branch": true, "claimed_at": true, "blocked_by": true,
}

func parseList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil
	}
	raw = strings.TrimPrefix(raw, "[")
	raw = strings.TrimSuffix(raw, "]")
	var out []string
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		item = strings.Trim(item, "`\"")
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

// ParseTaskQueueText parses task_queue.md content (used for both the worktree
// file and the base-ref blob when diffing control files).
func ParseTaskQueueText(data string) (packets []TaskPacket, summary map[string]string, err error) {
	summary = map[string]string{}
	var cur *TaskPacket
	var section string
	bodyLine := func(s *string, line string) { *s += line + "\n" }

	flush := func() {
		if cur != nil {
			packets = append(packets, *cur)
			cur = nil
		}
	}

	sc := bufio.NewScanner(strings.NewReader(data))
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), " \t")

		if m := summaryRe.FindStringSubmatch(line); m != nil {
			summary[m[1]] = m[2]
			continue
		}
		if m := packetHeadRe.FindStringSubmatch(line); m != nil {
			flush()
			cur = &TaskPacket{ID: m[1]}
			section = ""
			continue
		}
		if cur == nil {
			continue
		}
		if strings.HasPrefix(line, "# ") {
			flush()
			continue
		}
		cur.Raw += line + "\n"
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "## ") || strings.HasPrefix(trim, "### ") {
			name := strings.ToLower(strings.TrimSpace(strings.TrimLeft(trim, "#")))
			switch {
			case strings.HasPrefix(name, "change"):
				section = "change"
			case strings.HasPrefix(name, "acceptance"):
				section = "acceptance"
			case strings.HasPrefix(name, "tests"):
				section = "tests"
			default:
				section = ""
			}
			continue
		}
		if fm := fieldRe.FindStringSubmatch(trim); fm != nil {
			key, val := fm[1], fm[2]
			switch key {
			case "id":
				cur.ID = strings.Trim(val, "`\"")
			case "status":
				cur.Status = strings.Trim(val, "`\"")
			case "claimed_by":
				cur.ClaimedBy = strings.Trim(val, "`\"")
			case "branch":
				cur.Branch = strings.Trim(val, "`\"")
			case "claimed_at":
				cur.ClaimedAt = strings.Trim(val, "`\"")
			case "blocked_by":
				cur.BlockedBy = strings.Trim(val, "`\"")
			case "specs":
				cur.Specs = parseList(val)
			case "adrs":
				cur.ADRs = parseList(val)
			case "depends_on":
				cur.DependsOn = parseList(val)
			case "owned_paths":
				cur.OwnedPaths = parseList(val)
			case "forbidden_paths":
				cur.ForbiddenPaths = parseList(val)
			case "contract_inputs":
				cur.ContractInputs = parseList(val)
			case "contract_outputs":
				cur.ContractOutputs = parseList(val)
			case "consumers_checked":
				cur.ConsumersChecked = parseList(val)
			case "evidence_location":
				cur.EvidenceLocation = strings.Trim(val, "`\"")
			case "required_evidence":
				cur.RequiredEvidence = strings.Trim(val, "`\"")
			case "generated_artifacts", "cleanup_obligations":
				// recorded for completeness; not required by Q0
			default:
				appendSectionLine(section, cur, line, bodyLine)
			}
			continue
		}
		appendSectionLine(section, cur, line, bodyLine)
	}
	if err := sc.Err(); err != nil {
		return nil, nil, err
	}
	flush()
	return packets, summary, nil
}

func appendSectionLine(section string, cur *TaskPacket, line string, bodyLine func(*string, string)) {
	switch section {
	case "change":
		bodyLine(&cur.ChangeBody, line)
	case "acceptance":
		bodyLine(&cur.AcceptanceBody, line)
	case "tests":
		bodyLine(&cur.TestsBody, line)
	}
}

// ParseTaskQueue reads docs/10_implementation/task_queue.md from root.
func ParseTaskQueue(root string) ([]TaskPacket, map[string]string, error) {
	data, err := os.ReadFile(filepath.Join(root, "docs", "10_implementation", "task_queue.md"))
	if err != nil {
		return nil, nil, err
	}
	return ParseTaskQueueText(string(data))
}

// TaskPackets returns packets keyed by ID.
func TaskPackets(root string) (map[string]TaskPacket, error) {
	packets, _, err := ParseTaskQueue(root)
	if err != nil {
		return nil, err
	}
	m := map[string]TaskPacket{}
	for _, p := range packets {
		m[p.ID] = p
	}
	return m, nil
}

// TestFileRefs extracts file paths named in a packet's ## Tests section.
func (p TaskPacket) TestFileRefs() []string {
	var out []string
	for _, m := range testFileRe.FindAllString(p.TestsBody, -1) {
		if !strings.Contains(m, "TODO") {
			out = append(out, m)
		}
	}
	return out
}

// FieldValue returns one packet field's display value by key name.
func (p TaskPacket) FieldValue(key string) string {
	switch key {
	case "status":
		return p.Status
	case "claimed_by":
		return p.ClaimedBy
	case "branch":
		return p.Branch
	case "claimed_at":
		return p.ClaimedAt
	case "blocked_by":
		return p.BlockedBy
	}
	return ""
}

// packetFieldMap returns every `key: value` field line of a packet with its
// value, for field-level diffs.
func packetFieldMap(p TaskPacket) map[string]string {
	out := map[string]string{}
	sc := bufio.NewScanner(strings.NewReader(p.Raw))
	for sc.Scan() {
		trim := strings.TrimSpace(sc.Text())
		if fm := fieldRe.FindStringSubmatch(trim); fm != nil {
			out[fm[1]] = fm[2]
		}
	}
	return out
}

// pathsOverlap reports whether two repo-relative owned paths share a
// directory boundary — equal, or one strictly inside the other.
func pathsOverlap(a, b string) bool {
	a = pathpkg.Clean(a)
	b = pathpkg.Clean(b)
	if a == b {
		return true
	}
	if a == "." || b == "." {
		return true
	}
	return strings.HasPrefix(a, strings.TrimSuffix(b, "/")+"/") ||
		strings.HasPrefix(b, strings.TrimSuffix(a, "/")+"/")
}
