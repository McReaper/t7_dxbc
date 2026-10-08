//go:build windows

package d3dc

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCompileAndDisassemble(t *testing.T) {
	if !Available() {
		t.Skip("d3dcompiler_47.dll not found")
	}
	src := []byte("float4 tint; float4 main(float4 c : COLOR0) : SV_Target { return c * tint; }")
	code, err := Compile(src, filepath.Join(t.TempDir(), "t.hlsl"), "main", "ps_5_0")
	if err != nil {
		t.Fatal(err)
	}
	if string(code[:4]) != "DXBC" {
		t.Fatalf("not DXBC: %q", code[:4])
	}
	asm, err := Disassemble(code)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(asm, "ps_5_0") || !strings.Contains(asm, "mul o0.xyzw, v0.xyzw, cb0[0].xyzw") {
		t.Errorf("unexpected disassembly:\n%s", asm)
	}
	if _, err := Compile([]byte("float4 main() : SV_Target { return undefinedThing; }"), "bad.hlsl", "main", "ps_5_0"); err == nil ||
		!strings.Contains(err.Error(), "undeclared identifier") {
		t.Errorf("a compile error must carry the compiler's message: %v", err)
	}
}
