package taskgraph

import (
	"fmt"
	"regexp"
	"strings"

	"thinhthan/internal/conformance/gates"
)

// OwnedFile reports whether f is owned by the packet: f inside an owned path,
// or f = P.meta for an owned P, or f = the .meta of a folder first created by
// the packet (the folder chain down to an owned path) — repository_layout.md
// § Ownership Rules, ADR-0072.
func OwnedFile(p gates.TaskPacket, f string) bool {
	candidates := []string{f}
	if strings.HasSuffix(f, ".meta") {
		candidates = append(candidates, strings.TrimSuffix(f, ".meta"))
	}
	for _, o := range p.OwnedPaths {
		oc := cleanSlash(o)
		for _, c := range candidates {
			cc := cleanSlash(c)
			if cc == oc || strings.HasPrefix(cc, oc+"/") {
				return true
			}
			// .meta of a parent folder of an owned path
			if strings.HasSuffix(f, ".meta") && strings.HasPrefix(oc, cc+"/") {
				return true
			}
		}
	}
	return false
}

func cleanSlash(s string) string {
	s = strings.TrimSpace(s)
	parts := strings.Split(s, "/")
	var out []string
	for _, p := range parts {
		if p == "" || p == "." {
			continue
		}
		out = append(out, p)
	}
	return strings.Join(out, "/")
}

// pathsOverlap reports whether two repo paths share a directory boundary.
func pathsOverlap(a, b string) bool {
	a, b = cleanSlash(a), cleanSlash(b)
	return a == b || strings.HasPrefix(a, b+"/") || strings.HasPrefix(b, a+"/")
}

// OwnedForbiddenProblems enforces two path rules:
//  1. a packet may not own a path it also forbids (owned ∩ forbidden = ∅);
//  2. two packets may share an owned path only when depends_on orders them
//     (the later packet depends on the earlier owner) — rule 3 of §4.
func OwnedForbiddenProblems(packets []gates.TaskPacket) []string {
	var problems []string
	for _, p := range packets {
		for _, o := range p.OwnedPaths {
			for _, f := range p.ForbiddenPaths {
				if pathsOverlap(o, f) {
					problems = append(problems, p.ID+": owned path "+o+" overlaps forbidden path "+f)
				}
			}
		}
	}
	reach := depReachability(packets)
	for i := 0; i < len(packets); i++ {
		for j := i + 1; j < len(packets); j++ {
			a, b := packets[i], packets[j]
			for _, oa := range a.OwnedPaths {
				for _, ob := range b.OwnedPaths {
					if !pathsOverlap(oa, ob) {
						continue
					}
					// ownership overlap is legal only if depends_on orders them
					if !reach[a.ID][b.ID] && !reach[b.ID][a.ID] {
						problems = append(problems, fmt.Sprintf(
							"%s and %s share owned path %s ~ %s without a depends_on ordering",
							a.ID, b.ID, oa, ob))
					}
				}
			}
		}
	}
	return dedup(problems)
}

// depReachability returns reach[a][b] = b is reachable from a via depends_on.
func depReachability(packets []gates.TaskPacket) map[string]map[string]bool {
	byID := map[string]gates.TaskPacket{}
	for _, p := range packets {
		byID[p.ID] = p
	}
	reach := map[string]map[string]bool{}
	var walk func(id string, seen map[string]bool)
	walk = func(id string, seen map[string]bool) {
		p, ok := byID[id]
		if !ok {
			return
		}
		for _, d := range p.DependsOn {
			if seen[d] {
				continue
			}
			seen[d] = true
			walk(d, seen)
		}
	}
	for _, p := range packets {
		seen := map[string]bool{}
		walk(p.ID, seen)
		reach[p.ID] = seen
	}
	return reach
}

var (
	testFileRe   = regexp.MustCompile(`[A-Za-z0-9_./-]+\.(?:go|ps1|sh|cs|sql|json|md|asmdef|proto)`)
	blockerRefRe = regexp.MustCompile(`\(((?:BLK|OPS)-[0-9]+(?:\s*[,/]\s*(?:BLK|OPS)-[0-9]+)*)\)`)
	anyIDRe      = regexp.MustCompile(`(?:BLK|OPS)-[0-9]+`)
)

// TestPathProblems requires every repo-relative file path named in a
// packet's ## Tests to lie inside that packet's owned_paths (meta-implied
// ownership included). A path outside owned_paths is legal only when its
// bullet line carries an explicit (BLK-xxx)/(OPS-xxx) exemption marker that
// names an existing blocker entry — the spec-owner's recorded delegation.
func TestPathProblems(packets []gates.TaskPacket, knownIDs map[string]bool) []string {
	var problems []string
	for _, p := range packets {
		for _, line := range strings.Split(p.TestsBody, "\n") {
			for _, tok := range testFileRe.FindAllString(line, -1) {
				if !strings.Contains(tok, "/") {
					continue // bare file names are not paths
				}
				if OwnedFile(p, tok) {
					continue
				}
				m := blockerRefRe.FindStringSubmatch(line)
				if m == nil {
					problems = append(problems, fmt.Sprintf(
						"%s: test path %s is outside owned_paths and unmarked", p.ID, tok))
					continue
				}
				for _, id := range anyIDRe.FindAllString(m[1], -1) {
					if !knownIDs[id] {
						problems = append(problems, fmt.Sprintf(
							"%s: test path %s cites unknown blocker %s", p.ID, tok, id))
					}
				}
			}
		}
	}
	return dedup(problems)
}
