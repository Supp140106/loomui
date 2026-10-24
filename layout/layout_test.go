package layout

import (
	"reflect"
	"testing"
)

func req(min, grow, shrink uint8) Requirement {
	return Requirement{MinW: int(min), FlexGrowX: grow, FlexShrinkX: shrink}
}

func reqV(min, grow, shrink uint8) Requirement {
	return Requirement{MinH: int(min), FlexGrowY: grow, FlexShrinkY: shrink}
}

func TestDistribute(t *testing.T) {
	tests := []struct {
		name  string
		avail int
		reqs  []Requirement
		vert  bool
		want  []int
	}{
		{"no flex splits nothing", 10, []Requirement{req(3, 0, 0), req(4, 0, 0)}, false, []int{3, 4}},
		{"equal grow", 10, []Requirement{req(2, 1, 0), req(2, 1, 0)}, false, []int{5, 5}},
		{"weighted grow", 10, []Requirement{req(1, 1, 0), req(1, 2, 0)}, false, []int{3, 7}},
		{"partial grow", 6, []Requirement{req(2, 0, 0), req(2, 1, 0)}, false, []int{2, 4}},
		{"no grow leaves blank space", 10, []Requirement{req(2, 0, 0), req(2, 0, 0)}, false, []int{2, 2}},
		{"zero grow with one filler", 6, []Requirement{req(2, 0, 0), req(0, 1, 0)}, false, []int{2, 4}},
		{"even shrink", 4, []Requirement{req(3, 0, 1), req(3, 0, 1)}, false, []int{2, 2}},
		{"weighted shrink", 9, []Requirement{req(10, 0, 1), req(2, 0, 1)}, false, []int{8, 1}},
		{"never negative", 1, []Requirement{req(5, 0, 1), req(1, 0, 1)}, false, []int{1, 0}},
		{"empty request set", 4, nil, false, []int{}},
		{"single requirement", 8, []Requirement{req(3, 0, 0)}, false, []int{3}},
		{"exact fit", 5, []Requirement{req(5, 1, 0)}, false, []int{5}},
		{"no shrink factors still must fit", 2, []Requirement{req(3, 0, 0), req(3, 0, 0)}, false, []int{1, 1}},
		{"vertical equivalent", 4, []Requirement{reqV(3, 1, 0), reqV(3, 1, 0)}, true, []int{2, 2}},
		{"one cell share", 5, []Requirement{req(2, 1, 0), req(3, 1, 0)}, false, []int{2, 3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []int
			if tt.vert {
				got = VerticalSizes(tt.avail, tt.reqs)
			} else {
				got = HorizontalSizes(tt.avail, tt.reqs)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("distribute(%d) = %v, want %v", tt.avail, got, tt.want)
			}
			for _, g := range got {
				if g < 0 {
					t.Fatalf("negative size produced: %v", got)
				}
			}
		})
	}
}

func TestDistributeExtremeCandidates(t *testing.T) {
	for _, avail := range []int{-3, 0, 1, 2, 5, 13, 100} {
		for n := 0; n <= 3; n++ {
			reqs := make([]Requirement, n)
			for i := range reqs {
				reqs[i] = req(uint8(i), uint8(i%3), uint8(i%2))
			}
			got := HorizontalSizes(avail, reqs)
			for _, g := range got {
				if g < 0 {
					t.Fatalf("avail=%d n=%d: negative size %v", avail, n, got)
				}
			}
		}
	}
}
