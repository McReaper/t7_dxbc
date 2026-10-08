package dxbc

import (
	"fmt"
	"math/bits"
	"strings"
)

// alu describes an arithmetic or logic instruction: the type it reads its
// sources as, the type of its result, and how it combines them ($1, $2, $3
// stand for the sources, $N for the vector size of a conversion: int2).
type alu struct {
	in, out valType
	form    string
}

var alus = map[Opcode]alu{
	OpAdd: {tFloat, tFloat, "$1 + $2"}, OpMul: {tFloat, tFloat, "$1 * $2"}, OpDiv: {tFloat, tFloat, "$1 / $2"},
	OpMad: {tFloat, tFloat, "mad($1, $2, $3)"}, OpMin: {tFloat, tFloat, "min($1, $2)"},
	OpMax: {tFloat, tFloat, "max($1, $2)"},
	OpRcp: {tFloat, tFloat, "rcp($1)"}, OpRsq: {tFloat, tFloat, "rsqrt($1)"}, OpSqrt: {tFloat, tFloat, "sqrt($1)"},
	OpExp: {tFloat, tFloat, "exp2($1)"}, OpLog: {tFloat, tFloat, "log2($1)"}, OpFrc: {tFloat, tFloat, "frac($1)"},
	OpRoundNe: {tFloat, tFloat, "round($1)"}, OpRoundNi: {tFloat, tFloat, "floor($1)"},
	OpRoundPi: {tFloat, tFloat, "ceil($1)"}, OpRoundZ: {tFloat, tFloat, "trunc($1)"},
	OpDerivRtx: {tFloat, tFloat, "ddx($1)"}, OpDerivRty: {tFloat, tFloat, "ddy($1)"},
	OpDerivRtxCoarse: {tFloat, tFloat, "ddx_coarse($1)"}, OpDerivRtyCoarse: {tFloat, tFloat, "ddy_coarse($1)"},
	OpDerivRtxFine: {tFloat, tFloat, "ddx_fine($1)"}, OpDerivRtyFine: {tFloat, tFloat, "ddy_fine($1)"},
	OpLt: {tFloat, tUint, "$1 < $2 ? 0xffffffffu : 0u"}, OpGe: {tFloat, tUint, "$1 >= $2 ? 0xffffffffu : 0u"},
	OpEq: {tFloat, tUint, "$1 == $2 ? 0xffffffffu : 0u"}, OpNe: {tFloat, tUint, "$1 != $2 ? 0xffffffffu : 0u"},
	OpIlt: {tInt, tUint, "$1 < $2 ? 0xffffffffu : 0u"}, OpIge: {tInt, tUint, "$1 >= $2 ? 0xffffffffu : 0u"},
	OpIeq: {tInt, tUint, "$1 == $2 ? 0xffffffffu : 0u"}, OpIne: {tInt, tUint, "$1 != $2 ? 0xffffffffu : 0u"},
	OpUlt: {tUint, tUint, "$1 < $2 ? 0xffffffffu : 0u"}, OpUge: {tUint, tUint, "$1 >= $2 ? 0xffffffffu : 0u"},
	OpIadd: {tInt, tInt, "$1 + $2"}, OpImad: {tInt, tInt, "$1 * $2 + $3"}, OpImin: {tInt, tInt, "min($1, $2)"},
	OpImax: {tInt, tInt, "max($1, $2)"}, OpIneg: {tInt, tInt, "-$1"},
	OpIshl: {tInt, tInt, "$1 << $2"}, OpIshr: {tInt, tInt, "$1 >> $2"},
	OpUshr: {tUint, tUint, "$1 >> $2"}, OpAnd: {tUint, tUint, "$1 & $2"}, OpOr: {tUint, tUint, "$1 | $2"},
	OpXor: {tUint, tUint, "$1 ^ $2"}, OpNot: {tUint, tUint, "~$1"}, OpUmin: {tUint, tUint, "min($1, $2)"},
	OpUmax: {tUint, tUint, "max($1, $2)"}, OpUmad: {tUint, tUint, "$1 * $2 + $3"},
	OpCountbits: {tUint, tUint, "countbits($1)"}, OpFirstbitHi: {tUint, tUint, "dxbc_firstbit_hi($1)"},
	OpFirstbitLo: {tUint, tUint, "firstbitlow($1)"}, OpFirstbitShi: {tInt, tUint, "dxbc_firstbit_shi($1)"},
	OpBfrev: {tUint, tUint, "reversebits($1)"},
	OpFtoi:  {tFloat, tInt, "int$N($1)"}, OpFtou: {tFloat, tUint, "uint$N($1)"},
	OpItof: {tInt, tFloat, "float$N($1)"}, OpUtof: {tUint, tFloat, "float$N($1)"},
	OpF16tof32: {tUint, tFloat, "f16tof32($1)"}, OpF32tof16: {tFloat, tUint, "f32tof16($1)"},
}

// dots are the dot products: the components each source reads.
var dots = map[Opcode]int{OpDp2: 2, OpDp3: 3, OpDp4: 4}

// statements writes the program's instructions.
func (e *emitter) statements(body []Instruction) error {
	for i := range body {
		if err := e.statement(&body[i]); err != nil {
			return err
		}
	}
	return nil
}

func (e *emitter) statement(in *Instruction) error {
	e.current = in
	sat := in.Token&tokSaturate != 0
	if a, ok := alus[in.Op]; ok && len(in.Operands) >= 2 {
		e.aluStatement(in, a, sat)
		return nil
	}
	if n, ok := dots[in.Op]; ok && len(in.Operands) == 3 {
		d := &in.Operands[0]
		expr := fmt.Sprintf("dot(%s, %s)", e.read(&in.Operands[1], first(&in.Operands[1], n), tFloat),
			e.read(&in.Operands[2], first(&in.Operands[2], n), tFloat))
		e.write(d, expr, tFloat, sat)
		return nil
	}
	if f, ok := flowStatements[in.Op]; ok {
		f(e, in)
		return nil
	}
	if f, ok := otherStatements[in.Op]; ok {
		f(e, in, sat)
		return nil
	}
	e.unknown[mnemonic(in)] = true
	e.line("// %s", in.Disassemble())
	return nil
}

// aluStatement writes a destination = form(sources) instruction. A typeless
// mov copies bits: it reads and writes floats, which keeps them.
func (e *emitter) aluStatement(in *Instruction, a alu, sat bool) {
	d := &in.Operands[0]
	mask := destMask(d)
	expr := a.form
	if n := bits.OnesCount8(mask); n > 1 {
		expr = strings.ReplaceAll(expr, "$N", fmt.Sprint(n))
	} else {
		expr = strings.ReplaceAll(expr, "$N", "")
	}
	if name, _, ok := strings.Cut(a.form, "("); ok && strings.HasPrefix(name, "dxbc_") {
		e.helpers[name] = true
	}
	for i := len(in.Operands) - 1; i >= 1; i-- {
		src := e.read(&in.Operands[i], picks(&in.Operands[i], mask), a.in)
		expr = strings.ReplaceAll(expr, fmt.Sprintf("$%d", i), paren(src))
	}
	e.write(d, expr, a.out, sat)
}

// flowStatements are the control flow instructions.
var flowStatements = map[Opcode]func(*emitter, *Instruction){
	OpIf: func(e *emitter, in *Instruction) {
		e.line("if (%s) {", e.condition(in))
		e.depth++
	},
	OpElse: func(e *emitter, _ *Instruction) {
		e.depth--
		e.line("} else {")
		e.depth++
	},
	OpEndif:   closeBlock,
	OpLoop:    func(e *emitter, _ *Instruction) { e.line("while (true) {"); e.depth++ },
	OpEndloop: closeBlock,
	OpBreak:   func(e *emitter, _ *Instruction) { e.line("break;") },
	OpBreakc:  func(e *emitter, in *Instruction) { e.line("if (%s) break;", e.condition(in)) },
	OpContinue: func(e *emitter, _ *Instruction) {
		e.line("continue;")
	},
	OpContinuec: func(e *emitter, in *Instruction) { e.line("if (%s) continue;", e.condition(in)) },
	OpSwitch: func(e *emitter, in *Instruction) {
		e.line("switch (%s) {", e.read(&in.Operands[0], picks(&in.Operands[0], 1), tInt))
		e.depth++
	},
	OpCase: func(e *emitter, in *Instruction) {
		e.line("case %s:", e.read(&in.Operands[0], picks(&in.Operands[0], 1), tInt))
	},
	OpDefault:   func(e *emitter, _ *Instruction) { e.line("default:") },
	OpEndswitch: closeBlock,
	OpRet:       func(e *emitter, _ *Instruction) { e.line("return;") },
	OpRetc: func(e *emitter, in *Instruction) {
		e.line("if (%s) return;", e.condition(in))
	},
	OpDiscard: func(e *emitter, in *Instruction) { e.line("if (%s) discard;", e.condition(in)) },
}

func closeBlock(e *emitter, _ *Instruction) {
	e.depth--
	e.line("}")
}

// condition tests a conditional instruction's operand: _nz any bit set, _z
// none.
func (e *emitter) condition(in *Instruction) string {
	v := e.read(&in.Operands[0], picks(&in.Operands[0], 1), tUint)
	if in.Token&tokTestNonZero != 0 {
		return v + " != 0"
	}
	return v + " == 0"
}

// otherStatements are the instructions with more than one destination,
// helpers, or resources.
var otherStatements = map[Opcode]func(*emitter, *Instruction, bool){}

func init() {
	otherStatements[OpMov] = func(e *emitter, in *Instruction, sat bool) {
		d, src := &in.Operands[0], &in.Operands[1]
		cs := picks(src, destMask(d))
		if sat {
			e.write(d, e.read(src, cs, tFloat), tFloat, true)
			return
		}
		expr, t := e.readAny(src, cs)
		e.write(d, expr, t, false)
	}
	otherStatements[OpMovc] = func(e *emitter, in *Instruction, sat bool) {
		d := &in.Operands[0]
		mask := destMask(d)
		cond := e.read(&in.Operands[1], picks(&in.Operands[1], mask), tUint)
		a, b, t := e.readPair(&in.Operands[2], &in.Operands[3], mask, sat)
		e.write(d, fmt.Sprintf("%s ? %s : %s", paren(cond), paren(a), paren(b)), t, sat)
	}
	otherStatements[OpSincos] = func(e *emitter, in *Instruction, sat bool) {
		var exprs [2]string
		for i, fn := range []string{"sin", "cos"} {
			if d := &in.Operands[i]; d.Type != OperandNull {
				exprs[i] = fn + "(" + e.read(&in.Operands[2], picks(&in.Operands[2], destMask(d)), tFloat) + ")"
			}
		}
		e.writeBoth(in, exprs, tFloat, sat)
	}
	// imul/umul hi, lo: the high half is rarely asked for
	for op, t := range map[Opcode]valType{OpImul: tInt, OpUmul: tUint} {
		otherStatements[op] = func(e *emitter, in *Instruction, _ bool) {
			if in.Operands[0].Type != OperandNull {
				e.unknown[mnemonic(in)+" high half"] = true
			}
			d := &in.Operands[1]
			mask := destMask(d)
			a := e.read(&in.Operands[2], picks(&in.Operands[2], mask), t)
			b := e.read(&in.Operands[3], picks(&in.Operands[3], mask), t)
			e.write(d, paren(a)+" * "+paren(b), t, false)
		}
	}
	otherStatements[OpUdiv] = func(e *emitter, in *Instruction, _ bool) {
		var exprs [2]string
		for i, sym := range []string{"/", "%"} {
			if d := &in.Operands[i]; d.Type != OperandNull {
				mask := destMask(d)
				a := e.read(&in.Operands[2], picks(&in.Operands[2], mask), tUint)
				b := e.read(&in.Operands[3], picks(&in.Operands[3], mask), tUint)
				exprs[i] = paren(a) + " " + sym + " " + paren(b)
			}
		}
		e.writeBoth(in, exprs, tUint, false)
	}
	otherStatements[OpSwapc] = func(e *emitter, in *Instruction, _ bool) {
		// both results are computed before either is written: a destination
		// may be a source
		e.line("{")
		e.depth++
		types := [2]valType{}
		for i := range 2 {
			d := &in.Operands[i]
			mask := destMask(d)
			cond := e.read(&in.Operands[2], picks(&in.Operands[2], mask), tUint)
			a, b, t := e.readPair(&in.Operands[3+i], &in.Operands[4-i], mask, false)
			types[i] = t
			e.line("%s dxbc_swap%d = %s ? %s : %s;", vecType(t, bits.OnesCount8(mask)), i, paren(cond), paren(b), paren(a))
		}
		for i := range 2 {
			e.write(&in.Operands[i], fmt.Sprintf("dxbc_swap%d", i), types[i], false)
		}
		e.depth--
		e.line("}")
	}
	for op, fn := range map[Opcode]string{OpUbfe: "dxbc_ubfe", OpIbfe: "dxbc_ibfe", OpBfi: "dxbc_bfi"} {
		otherStatements[op] = func(e *emitter, in *Instruction, _ bool) { e.bitfield(in, fn) }
	}
	initResourceStatements()
}

// bitfield writes ubfe, ibfe and bfi component by component through the
// helpers the decompiled shader carries: HLSL has no bitfield intrinsics.
func (e *emitter) bitfield(in *Instruction, fn string) {
	e.helpers[fn] = true
	d := &in.Operands[0]
	t := tUint
	if fn == "dxbc_ibfe" {
		t = tInt
	}
	mask := destMask(d)
	var parts []string
	for i := range 4 {
		if mask&(1<<i) == 0 {
			continue
		}
		one := uint8(1 << i)
		args := make([]string, 0, 4)
		for j := 1; j < len(in.Operands); j++ {
			at := tUint
			if j == len(in.Operands)-1 && t == tInt {
				at = tInt
			}
			args = append(args, e.read(&in.Operands[j], picks(&in.Operands[j], one), at))
		}
		parts = append(parts, fn+"("+strings.Join(args, ", ")+")")
	}
	// one statement: every component reads its sources before any is written
	expr := parts[0]
	if len(parts) > 1 {
		expr = vecType(t, len(parts)) + "(" + strings.Join(parts, ", ") + ")"
	}
	e.write(d, expr, t, false)
}

// writeBoth writes the two results of sincos or udiv. When the first
// destination is also a source, the first result is kept aside before the
// second is computed.
func (e *emitter) writeBoth(in *Instruction, exprs [2]string, t valType, sat bool) {
	d0, d1 := &in.Operands[0], &in.Operands[1]
	if exprs[0] == "" || exprs[1] == "" || !aliases(d0, in.Operands[2:]) {
		for i, d := range []*Operand{d0, d1} {
			if exprs[i] != "" {
				e.write(d, exprs[i], t, sat)
			}
		}
		return
	}
	e.line("{")
	e.depth++
	e.line("%s dxbc_first = %s;", vecType(t, bits.OnesCount8(destMask(d0))), exprs[0])
	e.write(d1, exprs[1], t, sat)
	e.write(d0, "dxbc_first", t, sat)
	e.depth--
	e.line("}")
}

// aliases reports whether a destination register is one of the sources.
func aliases(dst *Operand, srcs []Operand) bool {
	if dst.Type != OperandTemp && dst.Type != OperandIndexableTemp {
		return false
	}
	var reads func(op *Operand) bool
	reads = func(op *Operand) bool {
		if op.Type == dst.Type && len(op.Index) > 0 && len(dst.Index) > 0 && op.Index[0].Imm == dst.Index[0].Imm {
			return true
		}
		for _, i := range op.Index {
			if i.Relative != nil && reads(i.Relative) {
				return true
			}
		}
		return false
	}
	for i := range srcs {
		if reads(&srcs[i]) {
			return true
		}
	}
	return false
}

// helperSources are the functions a decompiled shader may need.
var helperSources = map[string]string{
	"dxbc_ubfe": "uint dxbc_ubfe(uint w, uint o, uint s) { w &= 31; o &= 31; if (w == 0) return 0; " +
		"if (w + o < 32) return (s << (32 - (w + o))) >> (32 - w); return s >> o; }",
	"dxbc_ibfe": "int dxbc_ibfe(uint w, uint o, int s) { w &= 31; o &= 31; if (w == 0) return 0; " +
		"if (w + o < 32) return (s << (32 - (w + o))) >> (32 - w); return s >> o; }",
	"dxbc_bfi": "uint dxbc_bfi(uint w, uint o, uint i, uint b) { w &= 31; o &= 31; " +
		"uint m = ((1u << w) - 1) << o; return ((i << o) & m) | (b & ~m); }",
	// firstbit_hi and firstbit_shi count from the most significant bit,
	// firstbithigh from the least
	"dxbc_firstbit_hi":  "uint dxbc_firstbit_hi(uint x) { uint b = firstbithigh(x); return b == 0xffffffffu ? b : 31 - b; }",
	"dxbc_firstbit_shi": "uint dxbc_firstbit_shi(int x) { int b = firstbithigh(x); return b == -1 ? 0xffffffffu : 31 - b; }",
}

// readPair reads the two values a movc or swapc chooses between, in the type
// they share, or as bits if they differ.
func (e *emitter) readPair(x, y *Operand, mask uint8, sat bool) (a, b string, t valType) {
	if sat {
		return e.read(x, picks(x, mask), tFloat), e.read(y, picks(y, mask), tFloat), tFloat
	}
	a, ta := e.readAny(x, picks(x, mask))
	b, tb := e.readAny(y, picks(y, mask))
	if ta == tb {
		return a, b, ta
	}
	return convert(a, ta, tUint), convert(b, tb, tUint), tUint
}
