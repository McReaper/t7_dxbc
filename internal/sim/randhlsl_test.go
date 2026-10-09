package sim

import (
	"fmt"
	"math/rand/v2"
	"strings"
)

// randomShader writes a random pixel shader: expressions of every kind of
// value (float, int and uint vectors and their swizzles), the operations and
// intrinsics between them, constant-buffer arrays read at run time, texture
// samples and loads, locals written component by component, branches, loops
// and a switch. Each operation is kept finite (no division by zero, no
// square root of a negative), so the original and its decompilation can be
// compared by value. The same seed writes the same shader.
func randomShader(seed uint64) string {
	g := &shaderGen{r: rand.New(rand.NewPCG(seed, seed^0x9E3779B97F4A7C15))}
	var body strings.Builder
	g.out = &body
	g.locals = map[string]int{}
	for range 3 + g.r.IntN(4) {
		g.statement(1)
	}
	return fmt.Sprintf(`cbuffer Params : register(b0)
{
	float4 fv[4];
	uint4 uv[2];
	int4 iv[2];
	float fs;
	uint us;
};
Texture2D<float4> tex : register(t0);
Texture2D<uint4> ids : register(t1);
SamplerState smp : register(s0);

void main(float4 inF : TEXCOORD0, nointerpolation uint4 inU : TEXCOORD1, nointerpolation int4 inI : TEXCOORD2,
	out float4 outF : SV_Target0, out uint4 outU : SV_Target1)
{
	float4 lf = inF;
	uint4 lu = inU;
	int4 li = inI;
%s	outF = lf;
	outU = lu ^ asuint(li);
}
`, body.String())
}

type shaderGen struct {
	r      *rand.Rand
	out    *strings.Builder
	locals map[string]int
}

const (
	kFloat = iota
	kUint
	kInt
)

var kindNames = [...]string{"float", "uint", "int"}
var localOf = [...]string{"lf", "lu", "li"}

func (g *shaderGen) pick(xs ...string) string { return xs[g.r.IntN(len(xs))] }

// swizzle is a random swizzle of n components.
func (g *shaderGen) swizzle(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = "xyzw"[g.r.IntN(4)]
	}
	return string(b)
}

// mask is a random write mask: distinct components in order.
func (g *shaderGen) mask() string {
	var b []byte
	for _, c := range "xyzw" {
		if g.r.IntN(2) == 0 {
			b = append(b, byte(c))
		}
	}
	if len(b) == 0 {
		b = []byte{"xyzw"[g.r.IntN(4)]}
	}
	return string(b)
}

func vec(kind, n int) string {
	if n == 1 {
		return kindNames[kind]
	}
	return fmt.Sprintf("%s%d", kindNames[kind], n)
}

// leaf is a value read from somewhere: an input, a local, a constant, a
// literal, a texture.
func (g *shaderGen) leaf(kind, n int) string {
	sw := g.swizzle(n)
	switch g.r.IntN(6) {
	case 0:
		return [...]string{"inF", "inU", "inI"}[kind] + "." + sw
	case 1:
		return localOf[kind] + "." + sw
	case 2:
		idx := fmt.Sprint(g.r.IntN(2))
		if g.r.IntN(2) == 0 {
			idx = "(inU.x & 1)" // read at run time
		}
		switch kind {
		case kFloat:
			return fmt.Sprintf("fv[%s].%s", strings.ReplaceAll(idx, "1)", "3)"), sw)
		case kUint:
			return fmt.Sprintf("uv[%s].%s", idx, sw)
		}
		return fmt.Sprintf("iv[%s].%s", idx, sw)
	case 3:
		if kind == kFloat {
			return fmt.Sprintf("tex.SampleLevel(smp, inF.%s, %d).%s", g.swizzle(2), g.r.IntN(3), sw)
		}
		return fmt.Sprintf("%s(ids.Load(int3(inU.%s & 15, 0)).%s)", vec(kind, n), g.swizzle(2), sw)
	case 4:
		return g.literal(kind, n)
	}
	if kind == kFloat {
		return fmt.Sprintf("((%s)fs)", vec(kind, n)) // a cast broadcasts a scalar, a constructor doesn't
	}
	return fmt.Sprintf("((%s)us)", vec(kind, n))
}

func (g *shaderGen) literal(kind, n int) string {
	parts := make([]string, n)
	for i := range parts {
		switch kind {
		case kFloat:
			parts[i] = fmt.Sprintf("%.3f", g.r.Float64()*4-2)
		case kUint:
			parts[i] = fmt.Sprintf("%du", g.r.IntN(40))
		default:
			parts[i] = fmt.Sprint(g.r.IntN(40) - 20)
		}
	}
	return vec(kind, n) + "(" + strings.Join(parts, ", ") + ")"
}

// expr is a random expression of a kind and size.
func (g *shaderGen) expr(kind, n, depth int) string {
	if depth <= 0 || g.r.IntN(4) == 0 {
		return g.leaf(kind, n)
	}
	a := func() string { return g.expr(kind, n, depth-1) }
	switch kind {
	case kFloat:
		return g.floatExpr(n, depth, a)
	case kUint:
		return g.uintExpr(n, depth, a)
	}
	return g.intExpr(n, depth, a)
}

func (g *shaderGen) floatExpr(n, depth int, a func() string) string {
	switch g.r.IntN(16) {
	case 0:
		return fmt.Sprintf("(%s + %s)", a(), a())
	case 1:
		return fmt.Sprintf("(%s - %s)", a(), a())
	case 2:
		return fmt.Sprintf("(%s * %s)", a(), a())
	case 3:
		return fmt.Sprintf("(%s / (abs(%s) + 1))", a(), a())
	case 4:
		return fmt.Sprintf("mad(%s, %s, %s)", a(), a(), a())
	case 5:
		return fmt.Sprintf("%s(%s, %s)", g.pick("min", "max"), a(), a())
	case 6:
		return fmt.Sprintf("%s(%s)", g.pick("abs", "saturate", "frac", "floor", "ceil", "round", "trunc", "sin", "cos"), a())
	case 7:
		return fmt.Sprintf("sqrt(abs(%s))", a())
	case 8:
		return fmt.Sprintf("log2(abs(%s) + 1)", a())
	case 9:
		return fmt.Sprintf("exp2(clamp(%s, -4, 4))", a())
	case 10:
		return fmt.Sprintf("((%s)dot(%s, %s))", vec(kFloat, n), g.expr(kFloat, 3, depth-1), g.expr(kFloat, 3, depth-1))
	case 11:
		return fmt.Sprintf("(%s ? %s : %s)", g.condition(n, depth-1), a(), a())
	case 12:
		return fmt.Sprintf("%s(%s)", vec(kFloat, n), g.expr(g.pick2(kInt, kUint), n, depth-1))
	case 13:
		return fmt.Sprintf("lerp(%s, %s, saturate(%s))", a(), a(), a())
	case 14:
		return fmt.Sprintf("-(%s)", a())
	}
	return fmt.Sprintf("rsqrt(abs(%s) + 1)", a())
}

func (g *shaderGen) pick2(a, b int) int {
	if g.r.IntN(2) == 0 {
		return a
	}
	return b
}

func (g *shaderGen) uintExpr(n, depth int, a func() string) string {
	switch g.r.IntN(14) {
	case 0:
		return fmt.Sprintf("(%s + %s)", a(), a())
	case 1:
		return fmt.Sprintf("(%s * %s)", a(), a())
	case 2:
		return fmt.Sprintf("(%s / (%s | 1))", a(), a())
	case 3:
		return fmt.Sprintf("(%s %% (%s | 1))", a(), a())
	case 4:
		return fmt.Sprintf("(%s %s %s)", a(), g.pick("&", "|", "^"), a())
	case 5:
		return fmt.Sprintf("(%s %s (%s & 31))", a(), g.pick("<<", ">>"), a())
	case 6:
		return fmt.Sprintf("~%s", a())
	case 7:
		return fmt.Sprintf("%s(%s, %s)", g.pick("min", "max"), a(), a())
	case 8:
		return fmt.Sprintf("%s(%s)", g.pick("countbits", "reversebits", "firstbithigh", "firstbitlow"), a())
	case 9:
		return fmt.Sprintf("((%s >> 3) & 0x1F)", a()) // a bit field
	case 10:
		return fmt.Sprintf("%s(clamp(%s, 0, 1000))", vec(kUint, n), g.expr(kFloat, n, depth-1))
	case 11:
		return fmt.Sprintf("asuint(%s)", g.expr(kFloat, n, depth-1))
	case 12:
		return fmt.Sprintf("(%s ? %s : %s)", g.condition(n, depth-1), a(), a())
	}
	return fmt.Sprintf("%s(%s)", vec(kUint, n), g.expr(kInt, n, depth-1))
}

func (g *shaderGen) intExpr(n, depth int, a func() string) string {
	switch g.r.IntN(11) {
	case 0:
		return fmt.Sprintf("(%s + %s)", a(), a())
	case 1:
		return fmt.Sprintf("(%s - %s)", a(), a())
	case 2:
		return fmt.Sprintf("(%s * %s)", a(), a())
	case 3:
		return fmt.Sprintf("(%s >> (%s & 31))", a(), a())
	case 4:
		return fmt.Sprintf("%s(%s, %s)", g.pick("min", "max"), a(), a())
	case 5:
		return fmt.Sprintf("-(%s)", a())
	case 6:
		return fmt.Sprintf("abs(%s)", a())
	case 7:
		return fmt.Sprintf("firstbithigh(%s)", a())
	case 8:
		return fmt.Sprintf("%s(clamp(%s, -1000, 1000))", vec(kInt, n), g.expr(kFloat, n, depth-1))
	case 9:
		return fmt.Sprintf("(%s ? %s : %s)", g.condition(n, depth-1), a(), a())
	}
	return fmt.Sprintf("((%s << 7) >> 20)", a()) // a signed bit field
}

// condition is a comparison of n components.
func (g *shaderGen) condition(n, depth int) string {
	kind := g.r.IntN(3)
	return fmt.Sprintf("(%s %s %s)", g.expr(kind, n, depth), g.pick("<", "<=", ">", ">=", "==", "!="), g.expr(kind, n, depth))
}

// statement writes an assignment, a branch, a loop or a switch.
func (g *shaderGen) statement(depth int) {
	ind := strings.Repeat("\t", depth)
	switch k := g.r.IntN(8); {
	case k < 5 || depth > 2:
		kind := g.r.IntN(3)
		m := g.mask()
		fmt.Fprintf(g.out, "%s%s.%s = %s;\n", ind, localOf[kind], m, g.expr(kind, len(m), 3))
	case k == 5:
		fmt.Fprintf(g.out, "%sif (any(%s))\n%s{\n", ind, g.condition(1+g.r.IntN(2), 2), ind)
		g.statement(depth + 1)
		fmt.Fprintf(g.out, "%s}\n%selse\n%s{\n", ind, ind, ind)
		g.statement(depth + 1)
		fmt.Fprintf(g.out, "%s}\n", ind)
	case k == 6:
		v := fmt.Sprintf("k%d", depth)
		fmt.Fprintf(g.out, "%s[loop] for (uint %s = 0; %s < (inU.%s & 3); %s++)\n%s{\n", ind, v, v, g.swizzle(1), v, ind)
		g.statement(depth + 1)
		fmt.Fprintf(g.out, "%s\tif (lu.%s == %du) break;\n%s}\n", ind, g.swizzle(1), g.r.IntN(5), ind)
	default:
		fmt.Fprintf(g.out, "%sswitch (lu.%s & 3)\n%s{\n%scase 0:\n%scase 1:\n", ind, g.swizzle(1), ind, ind, ind)
		g.statement(depth + 1)
		fmt.Fprintf(g.out, "%s\tbreak;\n%scase 2:\n", ind, ind)
		g.statement(depth + 1)
		fmt.Fprintf(g.out, "%s\tbreak;\n%sdefault:\n%s\tbreak;\n%s}\n", ind, ind, ind, ind)
	}
}

// interpolationModes are the modifiers a random signature element gets.
var interpolationModes = []string{"", "", "nointerpolation ", "centroid ", "noperspective ", "noperspective centroid ",
	"sample ", "linear "}

// signatureElement is one random element: its declaration and an expression
// of its type built from a seed value.
func (g *shaderGen) signatureElement(i int, withMode bool) (decl, value string) {
	kind, n := g.r.IntN(3), 1+g.r.IntN(4)
	mode := ""
	switch {
	case kind != kFloat:
		mode = "nointerpolation " // integers are never interpolated
	case withMode:
		mode = g.pick(interpolationModes...)
	}
	decl = fmt.Sprintf("%s%s v%d : TEXCOORD%d", mode, vec(kind, n), i, i)
	switch kind {
	case kFloat:
		value = fmt.Sprintf("(%s)(p.%s * %d.5)", vec(kind, n), g.swizzle(n), i+1)
	case kUint:
		value = fmt.Sprintf("(%s)(id + %du)", vec(kind, n), i)
	default:
		value = fmt.Sprintf("(%s)(int(id) - %d)", vec(kind, n), i*3)
	}
	return decl, value
}

// randomVertexShader writes a vertex shader whose outputs are random
// elements, each with an interpolation mode its signature won't record.
func randomVertexShader(seed uint64) string {
	g := &shaderGen{r: rand.New(rand.NewPCG(seed, seed^0x2545F4914F6CDD1D))}
	var params, body []string
	for i := range 2 + g.r.IntN(8) {
		decl, value := g.signatureElement(i, true)
		params = append(params, "out "+decl)
		body = append(body, fmt.Sprintf("\tv%d = %s;", i, value))
	}
	if g.r.IntN(3) == 0 {
		params = append(params, "out float clip : SV_ClipDistance0")
		body = append(body, "\tclip = p.x - p.y;")
	}
	return fmt.Sprintf("void main(float4 p : POSITION, uint id : SV_VertexID, out float4 pos : SV_Position,\n\t%s)\n{\n\tpos = p;\n%s\n}\n",
		strings.Join(params, ",\n\t"), strings.Join(body, "\n"))
}

// randomPixelInputs writes a pixel shader whose inputs are random elements,
// some never read: their modes decided the packing all the same.
func randomPixelInputs(seed uint64) string {
	g := &shaderGen{r: rand.New(rand.NewPCG(seed, seed^0x9E3779B97F4A7C15))}
	params := []string{"float4 pos : SV_Position"}
	var sum []string
	for i := range 2 + g.r.IntN(8) {
		decl, _ := g.signatureElement(i, true)
		params = append(params, decl)
		if g.r.IntN(3) != 0 {
			sum = append(sum, fmt.Sprintf("dot(float4(v%d.xxxx), float4(1, 2, 3, 4))", i))
		}
	}
	if len(sum) == 0 {
		sum = []string{"0"}
	}
	return fmt.Sprintf("float4 main(%s) : SV_Target\n{\n\treturn pos * 0.001 + %s;\n}\n",
		strings.Join(params, ",\n\t"), strings.Join(sum, "\n\t\t+ "))
}
