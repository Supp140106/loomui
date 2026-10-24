package layout

// Rect is a screen region.
type Rect struct {
	X, Y, W, H int
}

// Requirement describes how an element wants to claim terminal space.
//
// Min sizes are the guaranteed lower bounds (may be exceeded when larger
// space is distributed). FlexGrow factors share leftover space after every
// minimum is satisfied; FlexShrink factors distribute a deficit when space
// is shorter than the combined minima.
type Requirement struct {
	MinW, MinH               int
	FlexGrowX, FlexGrowY     uint8
	FlexShrinkX, FlexShrinkY uint8
}
