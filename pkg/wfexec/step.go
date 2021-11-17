package wfexec

import (
	"context"
)

type (
	Steps []Step
	Step  interface {
		ID() uint64
		SetID(uint64)
		Exec(context.Context, *ExecRequest) (ExecResponse, error)
	}

	StepIdentifier struct{ id uint64 }

	execFn func(context.Context, *ExecRequest) (ExecResponse, error)

	genericStep struct {
		StepIdentifier
		fn execFn
	}
)

func (i *StepIdentifier) ID() uint64      { return i.id }
func (i *StepIdentifier) SetID(id uint64) { i.id = id }

func NewGenericStep(fn execFn) *genericStep {
	return &genericStep{fn: fn}
}

func (g *genericStep) Exec(ctx context.Context, r *ExecRequest) (ExecResponse, error) {
	return g.fn(ctx, r)
}

func (ss Steps) hash() map[Step]bool {
	out := make(map[Step]bool)
	for _, s := range ss {
		out[s] = true
	}

	return out
}

func (ss Steps) Contains(steps ...Step) bool {
	hash := ss.hash()
	for _, s1 := range steps {
		if !hash[s1] {
			return false
		}
	}

	return true
}

func (ss Steps) Exclude(steps ...Step) Steps {
	var (
		included = make([]Step, 0)
		excluded = Steps(steps).hash()
	)
	for _, s := range ss {
		if !excluded[s] {
			included = append(included, s)
		}
	}

	return included
}

func (ss Steps) IDs() []uint64 {
	if len(ss) == 0 {
		return nil
	}

	var ids = make([]uint64, len(ss))
	for i := range ss {
		ids[i] = ss[i].ID()
	}

	return ids
}
