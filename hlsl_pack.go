package dxbc

import "math/bits"

// interpolationChoices are the interpolation modifiers a decompiled element
// may get, the default first.
var interpolationChoices = []string{"", "nointerpolation ", "centroid ", "noperspective ", "noperspective centroid ",
	"sample ", "noperspective sample "}

// packing is the compiler's signature packer: elements, in declaration order,
// each go to the register of their interpolation group with the most free
// components that can take them (the lowest on a tie), or open the next
// register. A group is an interpolation mode, an integer element's being
// constant, or a system value of its own.
type packing []struct {
	group string
	free  int
}

// place packs an element of n components and returns its register and
// first component.
func (p *packing) place(group string, n int) (reg, comp int) {
	best := -1
	for i, r := range *p {
		if r.group == group && r.free >= n && (best < 0 || r.free > (*p)[best].free) {
			best = i
		}
	}
	if best < 0 {
		*p = append(*p, struct {
			group string
			free  int
		}{group, 4})
		best = len(*p) - 1
	}
	comp = 4 - (*p)[best].free
	(*p)[best].free -= n
	return best, comp
}

// packModes chooses each register's interpolation modifiers so that the
// compiler packs the elements into the registers and components they had. A
// vertex shader's output signature doesn't record the modes the original
// declared, nor a pixel shader's input signature those of the elements it
// never reads, yet they decided the packing. The elements are declared in
// register order; each register gets the first mode that puts its elements
// where they were and still lets every later register be packed so: a mode
// that packs this register can capture a later element of the same group.
// known holds the modes a pixel shader declares for the registers it reads.
func packModes(els []Element, known map[uint32]string) map[uint32]string {
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
	chosen := make([]string, len(regs))
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
			if next, ok := packRegister(regs[i], m, p); ok {
				chosen[i] = m
				if solve(i+1, next) {
					return true
				}
			}
		}
		return false
	}
	if !solve(0, nil) {
		greedyModes(regs, known, chosen)
	}
	modes := map[uint32]string{}
	for i, reg := range regs {
		modes[reg[0].Register] = chosen[i]
	}
	return modes
}

// greedyModes chooses, register by register, the first mode that packs it,
// or its first candidate: for a packing the search can't reproduce (an
// array, say).
func greedyModes(regs [][]Element, known map[uint32]string, chosen []string) {
	var p packing
	for i, reg := range regs {
		cands := packCandidates(reg, known)
		chosen[i] = cands[0]
		next, _ := packRegister(reg, cands[0], p)
		for _, m := range cands {
			if n, ok := packRegister(reg, m, p); ok {
				chosen[i], next = m, n
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
// and reports whether each lands where it was.
func packRegister(reg []Element, mode string, p packing) (packing, bool) {
	next := append(packing(nil), p...)
	ok := true
	for _, el := range reg {
		r, c := next.place(packGroup(mode, el), bits.OnesCount8(el.Mask))
		if r != int(el.Register) || c != bits.TrailingZeros8(el.Mask) {
			ok = false
		}
	}
	return next, ok
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
