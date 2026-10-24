package layout

// HorizontalSizes computes one slice of total==avail column widths for reqs.
func HorizontalSizes(avail int, reqs []Requirement) []int {
	return distribute(avail, reqs, false)
}

// VerticalSizes computes one slice of total==avail row heights for reqs.
func VerticalSizes(avail int, reqs []Requirement) []int {
	return distribute(avail, reqs, true)
}

// distribute flexes reqs along one axis: min sizes absorb leftover space in
// proportion to grow factors, or shrink in proportion to shrink factors
// (weighted by size) when space is short. Sizes are never negative.
func distribute(avail int, reqs []Requirement, vertical bool) []int {
	n := len(reqs)
	out := make([]int, n)
	if n == 0 || avail <= 0 {
		return out
	}

	mins := make([]int, n)
	sumMin := 0
	for i, r := range reqs {
		mins[i] = max(r.min(vertical), 0)
		sumMin += mins[i]
	}

	if sumMin <= avail {
		copy(out, mins)
		weights := make([]int, n)
		for i, r := range reqs {
			weights[i] = int(r.grow(vertical))
		}
		if sum(weights) == 0 {
			// no flex detected: minima stay, leftover space stays blank
			return out
		}
		grow := share(avail-sumMin, weights)
		for i := range out {
			out[i] += grow[i]
		}
		return out
	}

	deficit := sumMin - avail
	weights := make([]int, n)
	for i, r := range reqs {
		weights[i] = int(r.shrink(vertical)) * mins[i]
	}
	if sum(weights) == 0 {
		copy(weights, mins)
	}
	take := share(deficit, weights)
	for i := range out {
		out[i] = max(mins[i]-take[i], 0)
	}
	return out
}

// share splits pool across items proportional to their weights (0 = get
// nothing), returning per-item integer shares that sum exactly to pool.
func share(pool int, weights []int) []int {
	out := make([]int, len(weights))
	if pool <= 0 {
		return out
	}
	total := 0
	for _, w := range weights {
		total += max(w, 0)
	}
	if total == 0 {
		n := len(weights)
		for i := range out {
			out[i] = pool / n
		}
		return out
	}
	acc, run := 0, 0
	for i, w := range weights {
		acc += max(w, 0)
		p := acc * pool / total
		out[i] = p - run
		run = p
	}
	return out
}

func (r Requirement) min(vertical bool) int {
	if vertical {
		return r.MinH
	}
	return r.MinW
}

func (r Requirement) grow(vertical bool) uint8 {
	if vertical {
		return r.FlexGrowY
	}
	return r.FlexGrowX
}

func (r Requirement) shrink(vertical bool) uint8 {
	if vertical {
		return r.FlexShrinkY
	}
	return r.FlexShrinkX
}

func sum(v []int) int {
	s := 0
	for _, x := range v {
		s += x
	}
	return s
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
