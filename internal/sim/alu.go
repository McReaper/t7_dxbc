package sim

import (
	"math"
	"math/bits"

	"github.com/McReaper/t7dxbc"
)

// opFn runs an instruction that isn't control flow.
type opFn func(t *thread, in *t7dxbc.Instruction) error

// ops are the simulated instructions; resource ones are added by init.
var ops = map[t7dxbc.Opcode]opFn{}

func sat(in *t7dxbc.Instruction) bool { return in.Saturate() }

// float1, float2 and float3 build float instructions from what they do to
// one component.
func float1(f func(a float32) float32) opFn {
	return func(t *thread, in *t7dxbc.Instruction) error {
		a := t.fsrc(&in.Operands[1])
		var v Value
		for i := range v {
			v[i] = fbits(f(a[i]))
		}
		return t.store(&in.Operands[0], v, sat(in))
	}
}

func float2(f func(a, b float32) float32) opFn {
	return func(t *thread, in *t7dxbc.Instruction) error {
		a, b := t.fsrc(&in.Operands[1]), t.fsrc(&in.Operands[2])
		var v Value
		for i := range v {
			v[i] = fbits(f(a[i], b[i]))
		}
		return t.store(&in.Operands[0], v, sat(in))
	}
}

func float3(f func(a, b, c float32) float32) opFn {
	return func(t *thread, in *t7dxbc.Instruction) error {
		a, b, c := t.fsrc(&in.Operands[1]), t.fsrc(&in.Operands[2]), t.fsrc(&in.Operands[3])
		var v Value
		for i := range v {
			v[i] = fbits(f(a[i], b[i], c[i]))
		}
		return t.store(&in.Operands[0], v, sat(in))
	}
}

// fcmp builds a float comparison: all bits set when it holds.
func fcmp(f func(a, b float32) bool) opFn {
	return func(t *thread, in *t7dxbc.Instruction) error {
		a, b := t.fsrc(&in.Operands[1]), t.fsrc(&in.Operands[2])
		var v Value
		for i := range v {
			v[i] = mask32(f(a[i], b[i]))
		}
		return t.store(&in.Operands[0], v, false)
	}
}

func mask32(b bool) uint32 {
	if b {
		return 0xFFFFFFFF
	}
	return 0
}

// int1, int2 and int3 build integer instructions.
func int1(f func(a uint32) uint32) opFn {
	return func(t *thread, in *t7dxbc.Instruction) error {
		a := t.isrc(&in.Operands[1])
		var v Value
		for i := range v {
			v[i] = f(a[i])
		}
		return t.store(&in.Operands[0], v, false)
	}
}

func int2(f func(a, b uint32) uint32) opFn {
	return func(t *thread, in *t7dxbc.Instruction) error {
		a, b := t.isrc(&in.Operands[1]), t.isrc(&in.Operands[2])
		var v Value
		for i := range v {
			v[i] = f(a[i], b[i])
		}
		return t.store(&in.Operands[0], v, false)
	}
}

func int3(f func(a, b, c uint32) uint32) opFn {
	return func(t *thread, in *t7dxbc.Instruction) error {
		a, b, c := t.isrc(&in.Operands[1]), t.isrc(&in.Operands[2]), t.isrc(&in.Operands[3])
		var v Value
		for i := range v {
			v[i] = f(a[i], b[i], c[i])
		}
		return t.store(&in.Operands[0], v, false)
	}
}

// fmin and fmax return the number when the other is NaN.
func fmin(a, b float32) float32 {
	if a != a || b < a {
		return b
	}
	return a
}

func fmax(a, b float32) float32 {
	if a != a || b > a {
		return b
	}
	return a
}

func init() {
	for op, f := range map[t7dxbc.Opcode]func(a float32) float32{
		t7dxbc.OpFrc:  func(a float32) float32 { return a - float32(math.Floor(float64(a))) },
		t7dxbc.OpRcp:  func(a float32) float32 { return 1 / a },
		t7dxbc.OpRsq:  func(a float32) float32 { return float32(1 / math.Sqrt(float64(a))) },
		t7dxbc.OpSqrt: func(a float32) float32 { return float32(math.Sqrt(float64(a))) },
		t7dxbc.OpExp:  func(a float32) float32 { return float32(math.Exp2(float64(a))) },
		t7dxbc.OpLog:  func(a float32) float32 { return float32(math.Log2(float64(a))) },
		t7dxbc.OpRoundNe: func(a float32) float32 {
			return float32(math.RoundToEven(float64(a)))
		},
		t7dxbc.OpRoundNi:        func(a float32) float32 { return float32(math.Floor(float64(a))) },
		t7dxbc.OpRoundPi:        func(a float32) float32 { return float32(math.Ceil(float64(a))) },
		t7dxbc.OpRoundZ:         func(a float32) float32 { return float32(math.Trunc(float64(a))) },
		t7dxbc.OpDerivRtx:       zero1,
		t7dxbc.OpDerivRty:       zero1,
		t7dxbc.OpDerivRtxCoarse: zero1,
		t7dxbc.OpDerivRtyCoarse: zero1,
		t7dxbc.OpDerivRtxFine:   zero1,
		t7dxbc.OpDerivRtyFine:   zero1,
	} {
		ops[op] = float1(f)
	}
	for op, f := range map[t7dxbc.Opcode]func(a, b float32) float32{
		t7dxbc.OpAdd: func(a, b float32) float32 { return a + b },
		t7dxbc.OpMul: func(a, b float32) float32 { return a * b },
		t7dxbc.OpDiv: func(a, b float32) float32 { return a / b },
		t7dxbc.OpMin: fmin,
		t7dxbc.OpMax: fmax,
	} {
		ops[op] = float2(f)
	}
	ops[t7dxbc.OpMad] = float3(func(a, b, c float32) float32 { return float32(a*b) + c })
	for op, f := range map[t7dxbc.Opcode]func(a, b float32) bool{
		t7dxbc.OpEq: func(a, b float32) bool { return a == b },
		t7dxbc.OpNe: func(a, b float32) bool { return a != b },
		t7dxbc.OpLt: func(a, b float32) bool { return a < b },
		t7dxbc.OpGe: func(a, b float32) bool { return a >= b },
	} {
		ops[op] = fcmp(f)
	}
	for op, n := range map[t7dxbc.Opcode]int{t7dxbc.OpDp2: 2, t7dxbc.OpDp3: 3, t7dxbc.OpDp4: 4} {
		ops[op] = dot(n)
	}
	initInts()
	initMoves()
	initResources()
}

func zero1(float32) float32 { return 0 }

// dot builds dp2, dp3 and dp4: the sum, in order, of the first n products.
func dot(n int) opFn {
	return func(t *thread, in *t7dxbc.Instruction) error {
		a, b := t.fsrc(&in.Operands[1]), t.fsrc(&in.Operands[2])
		s := a[0] * b[0]
		for i := 1; i < n; i++ {
			s += float32(a[i] * b[i])
		}
		d := fbits(s)
		return t.store(&in.Operands[0], Value{d, d, d, d}, sat(in))
	}
}

func initInts() {
	for op, f := range map[t7dxbc.Opcode]func(a, b uint32) uint32{
		t7dxbc.OpIadd: func(a, b uint32) uint32 { return a + b },
		t7dxbc.OpAnd:  func(a, b uint32) uint32 { return a & b },
		t7dxbc.OpOr:   func(a, b uint32) uint32 { return a | b },
		t7dxbc.OpXor:  func(a, b uint32) uint32 { return a ^ b },
		t7dxbc.OpIshl: func(a, b uint32) uint32 { return a << (b & 31) },
		t7dxbc.OpIshr: func(a, b uint32) uint32 { return uint32(int32(a) >> (b & 31)) },
		t7dxbc.OpUshr: func(a, b uint32) uint32 { return a >> (b & 31) },
		t7dxbc.OpIeq:  func(a, b uint32) uint32 { return mask32(a == b) },
		t7dxbc.OpIne:  func(a, b uint32) uint32 { return mask32(a != b) },
		t7dxbc.OpIlt:  func(a, b uint32) uint32 { return mask32(int32(a) < int32(b)) },
		t7dxbc.OpIge:  func(a, b uint32) uint32 { return mask32(int32(a) >= int32(b)) },
		t7dxbc.OpUlt:  func(a, b uint32) uint32 { return mask32(a < b) },
		t7dxbc.OpUge:  func(a, b uint32) uint32 { return mask32(a >= b) },
		t7dxbc.OpImin: func(a, b uint32) uint32 { return uint32(min(int32(a), int32(b))) },
		t7dxbc.OpImax: func(a, b uint32) uint32 { return uint32(max(int32(a), int32(b))) },
		t7dxbc.OpUmin: func(a, b uint32) uint32 { return min(a, b) },
		t7dxbc.OpUmax: func(a, b uint32) uint32 { return max(a, b) },
	} {
		ops[op] = int2(f)
	}
	for op, f := range map[t7dxbc.Opcode]func(a uint32) uint32{
		t7dxbc.OpNot:         func(a uint32) uint32 { return ^a },
		t7dxbc.OpIneg:        func(a uint32) uint32 { return -a },
		t7dxbc.OpCountbits:   func(a uint32) uint32 { return uint32(bits.OnesCount32(a)) },
		t7dxbc.OpBfrev:       bits.Reverse32,
		t7dxbc.OpFirstbitLo:  firstbitLo,
		t7dxbc.OpFirstbitHi:  firstbitHi,
		t7dxbc.OpFirstbitShi: firstbitShi,
		t7dxbc.OpF16tof32:    func(a uint32) uint32 { return fbits(halfToFloat(uint16(a))) },
		t7dxbc.OpF32tof16:    func(a uint32) uint32 { return uint32(floatToHalf(math.Float32frombits(flush(a)))) },
		t7dxbc.OpItof:        func(a uint32) uint32 { return fbits(float32(int32(a))) },
		t7dxbc.OpUtof:        func(a uint32) uint32 { return fbits(float32(a)) },
		t7dxbc.OpFtoi:        func(a uint32) uint32 { return ftoi(math.Float32frombits(flush(a))) },
		t7dxbc.OpFtou:        func(a uint32) uint32 { return ftou(math.Float32frombits(flush(a))) },
	} {
		ops[op] = int1(f)
	}
	ops[t7dxbc.OpImad] = int3(func(a, b, c uint32) uint32 { return a*b + c })
	ops[t7dxbc.OpUmad] = int3(func(a, b, c uint32) uint32 { return a*b + c })
	ops[t7dxbc.OpUbfe] = int3(func(w, o, s uint32) uint32 { return ubfe(w, o, s) })
	ops[t7dxbc.OpIbfe] = int3(func(w, o, s uint32) uint32 { return ibfe(w, o, s) })
	ops[t7dxbc.OpBfi] = bfi
	ops[t7dxbc.OpImul] = twoResults(func(a, b uint32) (uint32, uint32) {
		p := int64(int32(a)) * int64(int32(b))
		return uint32(uint64(p) >> 32), uint32(p)
	})
	ops[t7dxbc.OpUmul] = twoResults(func(a, b uint32) (uint32, uint32) {
		hi, lo := bits.Mul32(a, b)
		return hi, lo
	})
	ops[t7dxbc.OpUdiv] = twoResults(func(a, b uint32) (uint32, uint32) {
		if b == 0 {
			return 0xFFFFFFFF, 0xFFFFFFFF
		}
		return a / b, a % b
	})
	ops[t7dxbc.OpUaddc] = twoResults(func(a, b uint32) (uint32, uint32) {
		s, c := bits.Add32(a, b, 0)
		return s, c
	})
	ops[t7dxbc.OpUsubb] = twoResults(func(a, b uint32) (uint32, uint32) {
		d, c := bits.Sub32(a, b, 0)
		return d, c
	})
}

// twoResults builds imul, umul, udiv, uaddc and usubb: two destinations, then
// two sources.
func twoResults(f func(a, b uint32) (uint32, uint32)) opFn {
	return func(t *thread, in *t7dxbc.Instruction) error {
		a, b := t.isrc(&in.Operands[2]), t.isrc(&in.Operands[3])
		var v0, v1 Value
		for i := range a {
			v0[i], v1[i] = f(a[i], b[i])
		}
		if err := t.store(&in.Operands[0], v0, false); err != nil {
			return err
		}
		return t.store(&in.Operands[1], v1, false)
	}
}

func bfi(t *thread, in *t7dxbc.Instruction) error {
	w, o := t.isrc(&in.Operands[1]), t.isrc(&in.Operands[2])
	ins, base := t.isrc(&in.Operands[3]), t.isrc(&in.Operands[4])
	var v Value
	for i := range v {
		v[i] = bfiBits(w[i], o[i], ins[i], base[i])
	}
	return t.store(&in.Operands[0], v, false)
}

// bfiBits inserts the low w bits of ins into base at bit o.
func bfiBits(w, o, ins, base uint32) uint32 {
	w, o = w&31, o&31
	m := (uint32(1)<<w - 1) << o
	return ins<<o&m | base&^m
}

func ubfe(w, o, s uint32) uint32 {
	w, o = w&31, o&31
	switch {
	case w == 0:
		return 0
	case w+o < 32:
		return s << (32 - (w + o)) >> (32 - w)
	}
	return s >> o
}

func ibfe(w, o, s uint32) uint32 {
	w, o = w&31, o&31
	switch {
	case w == 0:
		return 0
	case w+o < 32:
		return uint32(int32(s<<(32-(w+o))) >> (32 - w))
	}
	return uint32(int32(s) >> o)
}

func firstbitLo(a uint32) uint32 {
	if a == 0 {
		return 0xFFFFFFFF
	}
	return uint32(bits.TrailingZeros32(a))
}

// firstbitHi counts from the most significant bit.
func firstbitHi(a uint32) uint32 {
	if a == 0 {
		return 0xFFFFFFFF
	}
	return uint32(bits.LeadingZeros32(a))
}

// firstbitShi finds the first bit that differs from the sign, from the most
// significant.
func firstbitShi(a uint32) uint32 {
	if int32(a) < 0 {
		a = ^a
	}
	return firstbitHi(a)
}

func ftoi(f float32) uint32 {
	switch {
	case f != f:
		return 0
	case f >= 2147483647:
		return 0x7FFFFFFF
	case f <= -2147483648:
		return 0x80000000
	}
	return uint32(int32(f))
}

func ftou(f float32) uint32 {
	switch {
	case f != f || f <= 0:
		return 0
	case f >= 4294967295:
		return 0xFFFFFFFF
	}
	return uint32(f)
}

func initMoves() {
	ops[t7dxbc.OpMov] = func(t *thread, in *t7dxbc.Instruction) error {
		src := &in.Operands[1]
		if src.Modifier == t7dxbc.ModNone && !sat(in) {
			return t.store(&in.Operands[0], t.src(src), false)
		}
		f := t.fsrc(src)
		var v Value
		for i := range v {
			v[i] = math.Float32bits(f[i])
		}
		return t.store(&in.Operands[0], v, sat(in))
	}
	ops[t7dxbc.OpMovc] = func(t *thread, in *t7dxbc.Instruction) error {
		c, a, b := t.src(&in.Operands[1]), t.typeless(&in.Operands[2]), t.typeless(&in.Operands[3])
		var v Value
		for i := range v {
			v[i] = b[i]
			if c[i] != 0 {
				v[i] = a[i]
			}
		}
		return t.store(&in.Operands[0], v, sat(in))
	}
	ops[t7dxbc.OpSwapc] = func(t *thread, in *t7dxbc.Instruction) error {
		c, a, b := t.src(&in.Operands[2]), t.typeless(&in.Operands[3]), t.typeless(&in.Operands[4])
		var v0, v1 Value
		for i := range v0 {
			v0[i], v1[i] = a[i], b[i]
			if c[i] != 0 {
				v0[i], v1[i] = b[i], a[i]
			}
		}
		if err := t.store(&in.Operands[0], v0, false); err != nil {
			return err
		}
		return t.store(&in.Operands[1], v1, false)
	}
	ops[t7dxbc.OpSincos] = func(t *thread, in *t7dxbc.Instruction) error {
		a := t.fsrc(&in.Operands[2])
		var s, c Value
		for i := range a {
			s[i] = fbits(float32(math.Sin(float64(a[i]))))
			c[i] = fbits(float32(math.Cos(float64(a[i]))))
		}
		if err := t.store(&in.Operands[0], s, sat(in)); err != nil {
			return err
		}
		return t.store(&in.Operands[1], c, sat(in))
	}
	ops[t7dxbc.OpNop] = func(*thread, *t7dxbc.Instruction) error { return nil }
}

// typeless reads a movc or swapc source: bits, unless a modifier makes it a
// float.
func (t *thread) typeless(op *t7dxbc.Operand) Value {
	if op.Modifier == t7dxbc.ModNone {
		return t.src(op)
	}
	f := t.fsrc(op)
	var v Value
	for i := range v {
		v[i] = math.Float32bits(f[i])
	}
	return v
}

// halfToFloat and floatToHalf convert IEEE half floats, rounding to nearest
// even.
func halfToFloat(h uint16) float32 {
	sign := uint32(h>>15) << 31
	exp, man := uint32(h>>10&0x1F), uint32(h&0x3FF)
	switch {
	case exp == 0 && man == 0:
		return math.Float32frombits(sign)
	case exp == 0:
		return float32(math.Copysign(float64(man)/(1<<24), float64(int32(sign))))
	case exp == 0x1F:
		return math.Float32frombits(sign | 0xFF<<23 | man<<13)
	}
	return math.Float32frombits(sign | (exp+112)<<23 | man<<13)
}

func floatToHalf(f float32) uint16 {
	b := math.Float32bits(f)
	sign := uint16(b >> 16 & 0x8000)
	exp, man := int32(b>>23&0xFF), b&0x7FFFFF
	switch {
	case exp == 0xFF && man != 0:
		return sign | 0x7E00
	case exp == 0xFF || exp-112 >= 0x1F:
		return sign | 0x7C00
	case exp-112 <= 0:
		if exp < 103 {
			return sign
		}
		man |= 1 << 23
		shift := uint32(126 - exp)
		h := man >> shift
		rem := man & (1<<shift - 1)
		half := uint32(1) << (shift - 1)
		if rem > half || rem == half && h&1 != 0 {
			h++
		}
		return sign | uint16(h)
	}
	h := uint32(exp-112)<<10 | man>>13
	rem := man & 0x1FFF
	if rem > 0x1000 || rem == 0x1000 && h&1 != 0 {
		h++
	}
	return sign | uint16(h)
}
