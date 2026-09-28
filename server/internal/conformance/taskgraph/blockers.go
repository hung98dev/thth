package taskgraph

import (
	"regexp"
	"strings"
)

// BlockerEntry is one `### BLK-NNN` / `### OPS-NNN` entry of
// known_blockers.md.
type BlockerEntry struct {
	ID     string
	Blocks []string // task ids named by `blocks:` lines
	Raw    string
}

// BlockerDoc is known_blockers.md split into its Open and Resolved sections.
type BlockerDoc struct {
	Open     map[string]BlockerEntry
	Resolved map[string]BlockerEntry
}

var (
	blockerHeadRe = regexp.MustCompile(`^###\s+` + "`?" + `((?:BLK|OPS)-[0-9]+)` + "`?")
	blocksLineRe  = regexp.MustCompile(`^(?:(?:[-*+]|\d+[.)])\s*)?(?i:blocks):\s*(.*)$`)
	taskRefRe     = regexp.MustCompile(`\b((?:IMP|RMP)-[0-9]+)\b`)
)

// ParseBlockers parses known_blockers.md text. Entries under `## Open
// Blockers` land in Open, entries under `## Resolved Blockers` in Resolved.
func ParseBlockers(data string) BlockerDoc {
	doc := BlockerDoc{Open: map[string]BlockerEntry{}, Resolved: map[string]BlockerEntry{}}
	section := "" // "open" | "resolved"
	var cur *BlockerEntry
	flush := func() {
		if cur == nil {
			return
		}
		switch section {
		case "open":
			doc.Open[cur.ID] = *cur
		case "resolved":
			doc.Resolved[cur.ID] = *cur
		}
		cur = nil
	}
	for _, line := range strings.Split(data, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "## ") {
			flush()
			switch {
			case strings.Contains(trim, "Open Blockers"):
				section = "open"
			case strings.Contains(trim, "Resolved Blockers"):
				section = "resolved"
			default:
				section = ""
			}
			continue
		}
		if m := blockerHeadRe.FindStringSubmatch(trim); m != nil && (section == "open" || section == "resolved") {
			flush()
			cur = &BlockerEntry{ID: m[1]}
			continue
		}
		if cur == nil {
			continue
		}
		cur.Raw += line + "\n"
		if m := blocksLineRe.FindStringSubmatch(trim); m != nil {
			cur.Blocks = append(cur.Blocks, taskRefRe.FindAllString(m[1], -1)...)
		}
	}
	flush()
	return doc
}

// AllIDs returns every blocker id present in the document (open or resolved).
func (d BlockerDoc) AllIDs() map[string]bool {
	out := map[string]bool{}
	for id := range d.Open {
		out[id] = true
	}
	for id := range d.Resolved {
		out[id] = true
	}
	return out
}
