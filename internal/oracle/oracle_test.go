//go:build windows

package oracle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixtures(t *testing.T) map[string][]byte {
	t.Helper()
	paths, _ := filepath.Glob(filepath.Join("..", "..", "testdata", "*.dxbc"))
	if len(paths) < 2 {
		t.Fatal("no fixtures")
	}
	out := map[string][]byte{}
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		out[filepath.Base(p)] = b
	}
	return out
}

// A shader's interface equals its own, and differs from another's.
func TestDiffs(t *testing.T) {
	fx := fixtures(t)
	for name, b := range fx {
		if d := Diffs(b, b); len(d) > 0 {
			t.Errorf("%s against itself: %v", name, d)
		}
		if !OnlyDropped(b, b) {
			t.Errorf("%s against itself drops something", name)
		}
	}
	a, b := fx["ps_basic.dxbc"], fx["ps_math.dxbc"]
	d := Diffs(a, b)
	if len(d) == 0 || !strings.HasPrefix(d[0], "buffers:") {
		t.Errorf("ps_basic against ps_math: %v", d)
	}
	if OnlyDropped(a, b) || OnlyDropped(b, a) {
		t.Error("two different shaders differ by more than dropped reads")
	}
	if Diffs([]byte("not dxbc"), a)[0][:9] != "original:" || Diffs(a, []byte("x"))[0][:11] != "recompiled:" {
		t.Error("an unreadable shader is reported as such")
	}
}

func TestDroppedHelpers(t *testing.T) {
	if !subset([]string{"a"}, []string{"a", "b"}) || subset([]string{"c"}, []string{"a"}) || !subset(nil, nil) {
		t.Error("subset")
	}
	if got := withoutUse([]string{"  float4 x @0+16 used=true", "B kind=0"}); got[0] != "  float4 x @0+16" || got[1] != "B kind=0" {
		t.Errorf("withoutUse = %q", got)
	}
	orig := []string{"  float a @0+4 used=true", "  float b @4+4 used=false"}
	if gainsUse(orig, []string{"  float a @0+4 used=false", "  float b @4+4 used=false"}) {
		t.Error("losing a use is no gain")
	}
	if !gainsUse(orig, []string{"  float a @0+4 used=true", "  float b @4+4 used=true"}) {
		t.Error("b becomes used: a gain")
	}
}

func TestFirstDifference(t *testing.T) {
	cases := []struct {
		want, got []string
		says      string
	}{
		{[]string{"a", "b"}, []string{"a"}, "missing b"},
		{[]string{"a"}, []string{"a", "c"}, "extra c"},
		{[]string{"  float x @0+4 used=true"}, []string{"  float x @0+4 used=false"}, "use changed (used=true)"},
		{[]string{"a"}, []string{"b"}, "differs: a vs b"},
	}
	for _, c := range cases {
		if got := firstDifference(c.want, c.got); got != c.says {
			t.Errorf("firstDifference(%q, %q) = %q, want %q", c.want, c.got, got, c.says)
		}
	}
}
