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

// place packs an element of n components and returns its register.
func (p *packing) place(group string, n int) int {
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
	(*p)[best].free -= n
	return best
}

// packModes chooses each register's interpolation modifiers so that the
// compiler packs the elements into the registers they had. A vertex shader's
// output signature doesn't record the modes the original declared, nor a
// pixel shader's input signature those of the elements it never reads, yet
// they decided the packing. Declared in register order, a register's elements
// land in it when their group's other registers can't take them; known holds
// the modes a pixel shader declares for the registers it reads.
func packModes(els []Element, known map[uint32]string) map[uint32]string {
	modes := map[uint32]string{}
	var p packing
	for i := 0; i < len(els); {
		j := i
		for j < len(els) && els[j].Register == els[i].Register {
			j++
		}
		reg := els[i:j]
		i = j
		if reg[0].Register == NoRegister {
			continue
		}
		mode, next := choosePackMode(reg, known, p)
		modes[reg[0].Register] = mode
		p = next
	}
	return modes
}

// choosePackMode picks the mode that packs a register's elements into it:
// the one it declares, else the first that works, else the default.
func choosePackMode(reg []Element, known map[uint32]string, p packing) (string, packing) {
	first := reg[0]
	candidates := interpolationChoices
	if m, ok := known[first.Register]; ok || first.SystemValue != SVUndefined {
		candidates = []string{m}
	}
	var fallback packing
	fallbackMode := ""
	for _, m := range candidates {
		if first.Type != ComponentFloat && m != "nointerpolation " && len(candidates) > 1 {
			continue // an integer is never interpolated
		}
		try := append(packing(nil), p...)
		ok := true
		for _, el := range reg {
			if try.place(packGroup(m, el), bits.OnesCount8(el.Mask)) != int(first.Register) {
				ok = false
			}
		}
		if ok {
			return m, try
		}
		if fallback == nil {
			fallback, fallbackMode = try, m
		}
	}
	return fallbackMode, fallback
}

// packGroup is the interpolation group of an element with a mode.
func packGroup(mode string, el Element) string {
	switch {
	case el.SystemValue != SVUndefined:
		return "sv:" + el.SystemValue.String()
	case el.Type != ComponentFloat && mode == "":
		return "nointerpolation "
	}
	return mode
}
