package dxbc

import "math/bits"

// interpolationChoices are the interpolation modifiers a decompiled element
// may get, the default first.
var interpolationChoices = []string{"", "nointerpolation ", "centroid ", "noperspective ", "noperspective centroid ",
	"sample ", "noperspective sample "}

// packing is the compiler's signature packer: elements, in declaration order,
// each go to the register of their interpolation group with the most free
// components that can take them (the lowest on a tie), or open the next
// register. An array's elements each open a register of their own, which
// the elements declared after it may share. A group is an interpolation
// mode, an integer element's being constant, or a system value of its own.
type packing []struct {
	group string
	free  int
}

// worst is the register the compiler packs an element of n components into,
// or -1 for the next one.
func (p packing) worst(group string, n int) int {
	best := -1
	for i, r := range p {
		if r.group == group && r.free >= n && (best < 0 || r.free > p[best].free) {
			best = i
		}
	}
	return best
}

// put packs an element of n components into register at, or the next one
// when at is -1, and returns its register and first component.
func (p *packing) put(at int, group string, n int) (reg, comp int) {
	if at < 0 {
		*p = append(*p, struct {
			group string
			free  int
		}{group, 4})
		at = len(*p) - 1
	}
	comp = 4 - (*p)[at].free
	(*p)[at].free -= n
	return at, comp
}

// sigDecl declares signature elements with their interpolation modifiers:
// one element, or the elements of an array, at consecutive semantic indices.
type sigDecl struct {
	mode  string
	els   []Element
	array bool
}

// signatureLayout declares a signature's elements so that the compiler packs
// them into the registers and components they had: with the interpolation
// modes packModes chooses, an element the compiler would pack into an earlier
// register declared as an array, and arrays joined with the elements next to
// them they can take. known holds the modes a pixel shader declares for the
// registers it reads.
func signatureLayout(els []Element, known map[uint32]string) []sigDecl {
	modes, arrays := packModes(els, known)
	var decls []sigDecl
	for i, el := range els {
		if el.Register == NoRegister {
			continue
		}
		opens := i == 0 || els[i-1].Register != el.Register
		decls = append(decls, sigDecl{modes[el.Register], []Element{el}, opens && arrays[el.Register]})
	}
	return joinArrays(decls)
}

// plainLayout declares each element on its own: a vertex shader's inputs and
// a pixel shader's outputs, which aren't packed.
func plainLayout(els []Element) []sigDecl {
	var decls []sigDecl
	for _, el := range els {
		if el.Register != NoRegister {
			decls = append(decls, sigDecl{els: []Element{el}})
		}
	}
	return decls
}

// joinArrays joins an array with an element or array that continues it (the
// same semantic at the next index, in the next register, with the same
// components, type and mode) as long as the compiler still packs every
// element where it was: one array reads better than several.
func joinArrays(decls []sigDecl) []sigDecl {
	for joined := true; joined; {
		joined = false
		for i := 1; i < len(decls) && !joined; i++ {
			for j := range i {
				if !decls[i].array && !decls[j].array || !continues(decls[j], decls[i]) {
					continue
				}
				try := append([]sigDecl(nil), decls[:i]...)
				try[j] = sigDecl{decls[j].mode, append(append([]Element(nil), decls[j].els...), decls[i].els...), true}
				try = append(try, decls[i+1:]...)
				if packsAsDeclared(try) {
					decls, joined = try, true
					break
				}
			}
		}
	}
	return decls
}

// continues reports whether b's elements can follow a's in one array.
func continues(a, b sigDecl) bool {
	last, next := a.els[len(a.els)-1], b.els[0]
	return a.mode == b.mode && next.SystemValue == SVUndefined && last.SystemValue == SVUndefined &&
		next.Name == last.Name && next.Index == last.Index+1 && next.Register == last.Register+1 &&
		next.Mask == last.Mask && next.Type == last.Type
}

// packsAsDeclared reports whether the compiler packs declarations' elements
// where they were.
func packsAsDeclared(decls []sigDecl) bool {
	var p packing
	for _, d := range decls {
		for _, el := range d.els {
			group, n := packGroup(d.mode, el), bits.OnesCount8(el.Mask)
			at := -1
			if !d.array {
				at = p.worst(group, n)
			}
			if r, c := p.put(at, group, n); r != int(el.Register) || c != bits.TrailingZeros8(el.Mask) {
				return false
			}
		}
	}
	return true
}

// packModes chooses each register's interpolation modifiers so that the
// compiler packs the elements into the registers and components they had. A
// vertex shader's output signature doesn't record the modes the original
// declared, nor a pixel shader's input signature those of the elements it
// never reads, yet they decided the packing. The elements are declared in
// register order; each register gets the first mode that puts its elements
// where they were and still lets every later register be packed so: a mode
// that packs this register can capture a later element of the same group. A
// register whose first element the compiler would pack into an earlier
// register is opened by declaring that element an array: arrays says which.
func packModes(els []Element, known map[uint32]string) (modes map[uint32]string, arrays map[uint32]bool) {
	var regs [][]Element
	for i := 0; i < len(els); {
		j := i
		for j < len(els) && els[j].Register == els[i].Register {
			j++
		}
		if els[i].Register != NoRegister {
			regs = append(regs, els[i:j])
		}
		i = j
	}
	chosen, opened := make([]string, len(regs)), make([]bool, len(regs))
	budget := 1 << 14 // search steps; past it, each register takes what packs it alone
	var solve func(i int, p packing) bool
	solve = func(i int, p packing) bool {
		if i == len(regs) {
			return true
		}
		if budget--; budget < 0 {
			return false
		}
		for _, m := range packCandidates(regs[i], known) {
			if next, ok, array := packRegister(regs[i], m, p); ok {
				chosen[i], opened[i] = m, array
				if solve(i+1, next) {
					return true
				}
			}
		}
		return false
	}
	if !solve(0, nil) {
		greedyModes(regs, known, chosen, opened)
	}
	modes, arrays = map[uint32]string{}, map[uint32]bool{}
	for i, reg := range regs {
		modes[reg[0].Register] = chosen[i]
		if opened[i] {
			arrays[reg[0].Register] = true
		}
	}
	return modes, arrays
}

// greedyModes chooses, register by register, the first mode that packs it,
// or its first candidate: for a packing the search can't reproduce.
func greedyModes(regs [][]Element, known map[uint32]string, chosen []string, opened []bool) {
	var p packing
	for i, reg := range regs {
		cands := packCandidates(reg, known)
		chosen[i] = cands[0]
		next, _, array := packRegister(reg, cands[0], p)
		opened[i] = array
		for _, m := range cands {
			if n, ok, a := packRegister(reg, m, p); ok {
				chosen[i], next, opened[i] = m, n, a
				break
			}
		}
		p = next
	}
}

// packCandidates are the modes a register's elements may have: the one its
// declaration says, a system value's own, or any (constant only, when it
// holds an integer).
func packCandidates(reg []Element, known map[uint32]string) []string {
	first := reg[0]
	if m, ok := known[first.Register]; ok || first.SystemValue != SVUndefined {
		return []string{m}
	}
	for _, el := range reg {
		if el.Type != ComponentFloat {
			return []string{"nointerpolation "} // a register holding an integer is constant
		}
	}
	return interpolationChoices
}

// packRegister packs a register's elements with a mode, after what p holds,
// and reports whether each lands where it was, and whether its first element
// had to be an array to open the register: it opens it when the compiler
// would pack it into an earlier one.
func packRegister(reg []Element, mode string, p packing) (next packing, ok, array bool) {
	next = append(packing(nil), p...)
	ok = true
	for i, el := range reg {
		group, n := packGroup(mode, el), bits.OnesCount8(el.Mask)
		at := next.worst(group, n)
		if i == 0 && at >= 0 && el.SystemValue == SVUndefined && el.Mask&1 != 0 && int(el.Register) == len(next) {
			at, array = -1, true
		}
		if r, c := next.put(at, group, n); r != int(el.Register) || c != bits.TrailingZeros8(el.Mask) {
			ok = false
		}
	}
	return next, ok, array
}

// packGroup is the interpolation group of an element with a mode.
func packGroup(mode string, el Element) string {
	switch {
	case el.SystemValue != SVUndefined:
		return "sv:" + el.SystemValue.String()
	case el.Type != ComponentFloat:
		return "nointerpolation " // whatever its declaration says, an integer is constant
	}
	return mode
}
