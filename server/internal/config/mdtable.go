package config

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Cell is one parsed markdown-table cell: raw text with backticks stripped and
// both ends trimmed per content_authoring_contract.md §1.3.
type Row struct {
	// Line is the 1-based source line of this data row.
	Line int
	// Cells maps the exact lowercase header name to the cell text.
	Cells map[string]string
	// Order lists cell values in column order (canonical serialization).
	Order []string
	// Header is the table's column list in order.
	Header []string
}

// Table is one parsed markdown table.
type Table struct {
	// Line is the 1-based line of the header row.
	Line   int
	Header []string
	Rows   []Row
	// Section is the nearest preceding "##"-style heading text.
	Section string
}

// DocFile is one parsed markdown file: every table plus every heading that
// carries a backtick-quoted ID.
type DocFile struct {
	// RelPath is the repository-relative path.
	RelPath string
	Tables  []Table
	// HeadingIDs lists (line, id) for headings containing a backtick id.
	HeadingIDs []HeadingID
	// Lines holds the raw source for ad-hoc parsing (code blocks).
	Lines []string
}

// HeadingID is a backtick-quoted identifier in a markdown heading.
type HeadingID struct {
	Line int
	ID   string
}

var (
	backtickRe = regexp.MustCompile("`([^`]+)`")
	dividerRe  = regexp.MustCompile(`^\|?[\s:|-]*-[-\s:|]*\|?\s*$`)
)

// ParseMarkdown parses one catalog markdown file into tables + heading ids.
func ParseMarkdown(relPath, content string) *DocFile {
	f := &DocFile{RelPath: relPath}
	f.Lines = strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	var lastHeading string
	for i := 0; i < len(f.Lines); i++ {
		line := f.Lines[i]
		if strings.HasPrefix(line, "#") {
			lastHeading = strings.TrimSpace(strings.TrimLeft(line, "#"))
			for _, id := range backtickRe.FindAllStringSubmatch(line, -1) {
				f.HeadingIDs = append(f.HeadingIDs, HeadingID{Line: i + 1, ID: id[1]})
			}
			continue
		}
		// Table = row of pipes followed by a |---| divider.
		if !strings.HasPrefix(strings.TrimSpace(line), "|") || i+1 >= len(f.Lines) {
			continue
		}
		if !dividerRe.MatchString(strings.TrimSpace(f.Lines[i+1])) {
			continue
		}
		hdr := splitCells(line)
		t := Table{Line: i + 1, Header: hdr, Section: lastHeading}
		j := i + 2
		for j < len(f.Lines) && strings.HasPrefix(strings.TrimSpace(f.Lines[j]), "|") {
			cells := splitCells(f.Lines[j])
			row := Row{Line: j + 1, Cells: map[string]string{}, Order: cells, Header: hdr}
			for k, h := range hdr {
				if k < len(cells) {
					row.Cells[h] = cells[k]
				}
			}
			t.Rows = append(t.Rows, row)
			j++
		}
		f.Tables = append(f.Tables, t)
		i = j - 1
	}
	return f
}

// splitCells splits a "| a | b |" row into trimmed, backtick-stripped cells.
func splitCells(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	out := make([]string, len(parts))
	for i, p := range parts {
		p = strings.ReplaceAll(p, "`", "")
		out[i] = strings.TrimSpace(p)
	}
	return out
}

// TablesWithHeader returns every table in f whose header contains all the
// named columns (header names are matched exactly, case preserved).
func (f *DocFile) TablesWithHeader(cols ...string) []Table {
	var out []Table
	for _, t := range f.Tables {
		set := map[string]bool{}
		for _, h := range t.Header {
			set[h] = true
		}
		ok := true
		for _, c := range cols {
			if !set[c] {
				ok = false
				break
			}
		}
		if ok {
			out = append(out, t)
		}
	}
	return out
}

// LoadDocFile reads and parses docs/<rel> under root; missing files produce a
// CATALOG_FILE_NOT_FOUND diagnostic and a nil DocFile.
func LoadDocFile(root, rel string, ds *Diagnostics) *DocFile {
	p := filepath.Join(root, filepath.FromSlash(rel))
	data, err := os.ReadFile(p)
	if err != nil {
		ds.Add(rel, 0, 0, CodeCatalogFileNotFound, err.Error())
		return nil
	}
	return ParseMarkdown(rel, string(data))
}
