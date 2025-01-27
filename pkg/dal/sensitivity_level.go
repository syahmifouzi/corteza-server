package dal

type (
	SensitivityLevel struct {
		Handle string
		ID     uint64
		Level  int
	}
	SensitivityLevelSet []SensitivityLevel

	sensitivityLevelIndex struct {
		set SensitivityLevelSet

		byHandle map[string]int
		byID     map[uint64]int
	}
)

func SensitivityLevelIndex(levels ...SensitivityLevel) *sensitivityLevelIndex {
	out := &sensitivityLevelIndex{
		set:      make(SensitivityLevelSet, len(levels)),
		byHandle: make(map[string]int),
		byID:     make(map[uint64]int),
	}

	for i, l := range levels {
		out.set[i] = l
		out.byHandle[l.Handle] = i
		out.byID[l.ID] = i
	}

	return out
}

func (sli sensitivityLevelIndex) includes(l uint64) (ok bool) {
	if l == 0 {
		return true
	}

	if sli.byID == nil {
		return false
	}

	_, ok = sli.byID[l]
	return
}

func (sli sensitivityLevelIndex) isSubset(a, b uint64) (ok bool) {
	// Edgecases
	// If A is zero theneverything is possible
	if a == 0 {
		return true
	}
	// If B is zero, then A must also be zero
	if b == 0 {
		return a == 0
	}

	var lvlA, lvlB int

	if lvlA, ok = sli.byID[a]; !ok {
		return false
	}

	if lvlB, ok = sli.byID[b]; !ok {
		return false
	}

	return lvlA <= lvlB
}

func (ss SensitivityLevelSet) includes(l uint64) (ok bool) {
	for _, s := range ss {
		ok = ok || s.ID == l
	}

	return
}
