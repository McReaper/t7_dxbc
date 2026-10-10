//go:build windows

package dxbc

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/McReaper/dxbc/internal/d3dc"
)

// The compiler packs each case's declarations as its signature says: they
// are a vertex shader's outputs, packed as a pixel shader's inputs are.
func TestLayoutsCompile(t *testing.T) {
	if !d3dc.Available() {
		t.Skip("d3dcompiler_47.dll not found")
	}
	for _, c := range layoutCases {
		if c.els[1].SystemValue != SVUndefined {
			continue // no declaration reproduces it
		}
		var params, body []string
		for _, d := range signatureLayout(c.els, c.known) {
			params = append(params, "out "+d.declaration("o_"))
			name, typ := "o_"+fieldName(d.els[0]), elementHLSLType(d.els[0])
			for i := range d.els {
				if d.array {
					body = append(body, fmt.Sprintf("%s[%d] = (%s)p.x;", name, i, typ))
				} else {
					body = append(body, fmt.Sprintf("%s = (%s)p.x;", name, typ))
				}
			}
		}
		src := fmt.Sprintf("void main(float4 p : POSITION, %s) { %s }", strings.Join(params, ", "), strings.Join(body, " "))
		code, err := d3dc.Compile([]byte(src), "layout.hlsl", "main", "vs_5_0")
		if err != nil {
			t.Fatalf("%s: %v\n%s", c.name, err, src)
		}
		cont, err := Parse(code)
		if err != nil {
			t.Fatal(err)
		}
		_, out, err := cont.Signatures()
		if err != nil {
			t.Fatal(err)
		}
		if got, want := slots(out), slots(c.els); !reflect.DeepEqual(got, want) {
			t.Errorf("%s:\n got  %v\n want %v\n%s", c.name, got, want, src)
		}
	}
}

// slots lists where elements are, by semantic.
func slots(els []Element) []string {
	var out []string
	for _, el := range els {
		out = append(out, fmt.Sprintf("%s%d r%d %s", strings.ToUpper(el.Name), el.Index, el.Register, MaskString(el.Mask)))
	}
	sort.Strings(out)
	return out
}
