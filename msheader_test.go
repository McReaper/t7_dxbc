package dxbc

import (
	"regexp"
	"strconv"
	"strings"
)

// msHeader is what Microsoft's disassembly states in its reflection header:
// the oracle the RDEF and signature readers are checked against.
type msHeader struct {
	vars     []msVar
	bindings [][]string // the raw rows split on spaces: Name, Type, Format, Dim, HLSL Bind, Count
	inputs   [][]string // Name, Index, Mask, Register, SysValue, Format, Used
	outputs  [][]string
}

// msVar is a top-level variable of a constant buffer or a structured buffer's
// element layout, with its offset and size.
type msVar struct {
	buffer, name string
	offset, size uint32
	unused       bool
}

var (
	bufferLine = regexp.MustCompile(`^// (?:cbuffer|tbuffer) (\S+)$|^// Resource bind info for (\S+)$`)
	varLine    = regexp.MustCompile(`^//   (?:\S.*?\s|\} )(\$?[A-Za-z_]\w*)(?:\[\d+\])*;\s*// Offset:\s+(\d+) Size:\s+(\d+)( \[unused\])?$`)
)

func parseMSHeader(asm string) msHeader {
	var h msHeader
	lines := strings.Split(strings.ReplaceAll(asm, "\r\n", "\n"), "\n")
	buffer := ""
	for i := 0; i < len(lines); i++ {
		l := strings.TrimRight(lines[i], " ")
		if m := bufferLine.FindStringSubmatch(l); m != nil {
			buffer = m[1] + m[2]
			continue
		}
		if m := varLine.FindStringSubmatch(l); m != nil && buffer != "" {
			off, _ := strconv.Atoi(m[2])
			size, _ := strconv.Atoi(m[3])
			h.vars = append(h.vars, msVar{buffer, m[1], uint32(off), uint32(size), m[4] != ""})
			continue
		}
		switch l {
		case "// Resource Bindings:":
			// split on spaces: a name longer than its column pushes the rest
			// of the row along, and no field holds a space
			var raw []string
			_, raw, i = msTable(lines, i)
			for _, r := range raw {
				h.bindings = append(h.bindings, strings.Fields(strings.TrimPrefix(r, "//")))
			}
		case "// Input signature:":
			h.inputs, _, i = msTable(lines, i)
		case "// Output signature:":
			h.outputs, _, i = msTable(lines, i)
		}
	}
	return h
}

// msTable reads the fixed-width table after a title: the column widths come
// from its line of dashes, its rows run to the next blank comment line. It
// returns the rows cut into columns, and as they are.
func msTable(lines []string, i int) ([][]string, []string, int) {
	for i++; i < len(lines) && !strings.HasPrefix(lines[i], "// ---"); i++ {
		// "// no Input": a table that isn't there; code: the header is over
		if strings.HasPrefix(lines[i], "// no ") || !strings.HasPrefix(lines[i], "//") {
			return nil, nil, i
		}
	}
	if i == len(lines) {
		return nil, nil, i
	}
	var cols [][2]int
	dash := lines[i]
	for j := 3; j < len(dash); {
		if dash[j] != '-' {
			j++
			continue
		}
		k := j
		for k < len(dash) && dash[k] == '-' {
			k++
		}
		cols = append(cols, [2]int{j, k})
		j = k
	}
	var rows [][]string
	var raw []string
	for i++; i < len(lines); i++ {
		l := strings.TrimRight(lines[i], "\r")
		if !strings.HasPrefix(l, "//") || strings.TrimSpace(strings.TrimPrefix(l, "//")) == "" {
			break
		}
		if strings.HasPrefix(l, "// no ") { // "// no Input": the table is empty
			return nil, nil, i
		}
		raw = append(raw, l)
		row := make([]string, len(cols))
		for c, span := range cols {
			end := span[1]
			if c == len(cols)-1 || end > len(l) {
				end = len(l)
			}
			if span[0] < len(l) {
				row[c] = l[span[0]:end]
			}
		}
		rows = append(rows, row)
	}
	return rows, raw, i
}
