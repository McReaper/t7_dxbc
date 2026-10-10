package t7dxbc

import (
	"strings"
	"testing"
)

// msInstructions are the instruction lines of Microsoft's disassembly, after
// the version line: trimmed, comments dropped, an immediate constant buffer's
// rows joined into its one declaration.
func msInstructions(asm string) []string {
	var out []string
	code := false
	icb := ""
	for _, l := range strings.Split(strings.ReplaceAll(asm, "\r\n", "\n"), "\n") {
		l = strings.TrimSpace(l)
		if !code {
			code = isVersionLine(l)
			continue
		}
		if l == "" || strings.HasPrefix(l, "//") {
			continue
		}
		if icb != "" || strings.HasPrefix(l, "dcl_immediateConstantBuffer") {
			icb += " " + l
			if strings.Count(icb, "{") == strings.Count(icb, "}") {
				out = append(out, normalizeLine(icb))
				icb = ""
			}
			continue
		}
		out = append(out, normalizeLine(l))
	}
	return out
}

func isVersionLine(l string) bool {
	for _, s := range []string{"ps_", "vs_", "gs_", "hs_", "ds_", "cs_"} {
		if strings.HasPrefix(l, s) && len(l) == 6 {
			return true
		}
	}
	return false
}

func normalizeLine(l string) string { return strings.Join(strings.Fields(l), " ") }

// disasmDiffs compares the decoder's disassembly of a shader with
// Microsoft's: the first differing lines, and the mnemonic of each.
func disasmDiffs(code []byte, asm string) (diffs []string, mnemonics []string) {
	c, err := Parse(code)
	if err != nil {
		return []string{err.Error()}, []string{"parse"}
	}
	p, err := DecodeProgram(c.Chunk("SHEX", "SHDR"))
	if err != nil {
		return []string{err.Error()}, []string{"decode"}
	}
	got := p.Disassemble()
	for i := range got {
		got[i] = normalizeLine(got[i])
	}
	want := msInstructions(asm)
	for i := 0; i < len(got) || i < len(want); i++ {
		g, w := "", ""
		if i < len(got) {
			g = got[i]
		}
		if i < len(want) {
			w = want[i]
		}
		if g != w {
			diffs = append(diffs, "got  "+g+"\n  want "+w)
			mnemonics = append(mnemonics, strings.Fields(w + " ?")[0])
		}
	}
	return diffs, mnemonics
}

func TestDisassemblyMatchesMicrosoft(t *testing.T) {
	for name, f := range fixtures(t) {
		diffs, _ := disasmDiffs(f[0], string(f[1]))
		for i, d := range diffs {
			if i == 6 {
				t.Errorf("%s: …%d more", name, len(diffs)-i)
				break
			}
			t.Errorf("%s:\n  %s", name, d)
		}
	}
}
