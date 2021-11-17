package wfexec

import (
	"context"

	"github.com/cortezaproject/corteza-server/pkg/expr"
)

type (
	// Iterator can be returned from Exec fn as ExecResponse
	//
	// It helps session's exec fn() to properly navigate through graph
	// by calling is/break/iterator/next function
	//Iterator interface {
	//	// Is the given step this iterator step
	//	Is(Step) bool
	//
	//	// Initialize iterator
	//	Start(context.Context, *expr.Vars) error
	//
	//	// Break fn is called when loop is forcefully broken
	//	Break() Step
	//
	//	Iterator() Step
	//
	//	// Next is called before each iteration and returns
	//	// 1st step of the iteration branch and variables that are added to the scope
	//	Next(context.Context, *expr.Vars) (Step, *expr.Vars, error)
	//}

	ResultEvaluator interface {
		EvalResults(ctx context.Context, results *expr.Vars) (out *expr.Vars, err error)
	}

	// Handles communication between Session's exec() fn and iterator handler
	genericIterator struct {
		// 1st step of the iterated branch
		base Step

		// set when the first iteration is started via (exec)
		started bool

		// graph to use for iteration
		// note that generic iterator does NOT iterate the whole graph
		// but only one branch, starting from the base step
		g *Graph

		scope *expr.Vars

		h IteratorHandler
	}

	IteratorHandler interface {
		Start(context.Context, *expr.Vars) error
		More(context.Context, *expr.Vars) (bool, error)
		Next(context.Context, *expr.Vars) (*expr.Vars, error)
	}
)

const (
	DefaultMaxIteratorBufferSize uint = 1000
)

var (
	MaxIteratorBufferSize uint = DefaultMaxIteratorBufferSize

	_ iterator = &genericIterator{}
)

// GenericIterator creates a wrapper around IteratorHandler and
// returns genericIterator that implements Iterator interface
func GenericIterator(g *Graph, first Step, h IteratorHandler, scope *expr.Vars) *genericIterator {
	return &genericIterator{
		g:     g,
		base:  first,
		h:     h,
		scope: scope,
	}
}

// Exec on generic iterator is executed two times:
//  1. Signaling executor that a new block should be put on the stack
//     this is achieved by returning a block
//     Internally this is controlled by setting current step to the base step
//
//  2. After 1st exec, base loop step is executed
func (i *genericIterator) Exec(ctx context.Context, r *ExecRequest) (ExecResponse, error) {
	if !i.started {
		if err := i.h.Start(ctx, r.Scope); err != nil {
			return nil, err
		}

		i.started = true
		return i, nil
	}

	return i.base.Exec(ctx, r)
}

func (i *genericIterator) Continue() Step {
	return i.base
}

func (i *genericIterator) Children(c Step) Steps {
	return i.g.Children(c)
}

func (i *genericIterator) Exit() (Steps, *expr.Vars) {
	return nil, nil
}

//func (i *genericIterator) Is(s Step) bool                                { return i.iter == s }
//func (i *genericIterator) IsLoop() bool      { return true }
//func (i *genericIterator) CurrentStep() Step { return i.current }
//func (i *genericIterator) FirstStep() Step   { return i.iter }

// Next on iterator iterates through the branch in the given graph
//
// If iterator step (iter field) implements ResultEvaluator it calls
// EvalResults on it before returning it. If iterator step does not implement it,
// results are omitted.
func (i *genericIterator) Next(ctx context.Context, current Step) (next Steps, err error) {
	if i.started {
		panic("iterator not started")
	}

	next = i.g.Children(current)
	if len(next) > 0 {
		return
	}

	var is bool
	if is, err = i.h.More(context.TODO(), i.scope); err != nil {
		return nil, err
	} else if is {
		// gracefully handling last step of iteration branch
		// that does not point back to the iterator step
		//return Steps{i.FirstStep()}, nil

		if i.scope, err = i.h.Next(ctx, i.scope); err != nil {
			return
		}

		if re, is := i.base.(ResultEvaluator); is {
			if i.scope, err = re.EvalResults(ctx, i.scope); err != nil {
				return
			}
		}
	}

	return
}

//func (i *genericIterator) Start(ctx context.Context, s *expr.Vars) error { return i.h.Start(ctx, s) }
//func (i *genericIterator) Exec(ctx context.Context, r *ExecRequest) (ExecResponse, error) {
//	// @todo how do we know it's the last step and we need to go into next iteration
//	//if (&Graph{}).Children(i.current) == nil {
//	//i.nextIteration()
//	//}
//	return i, i.h.Start(ctx, r.Input)
//}

//func (i *genericIterator) Break() Step    { return i.exit }
//func (i *genericIterator) Exit() (Steps, *expr.Vars) { return nil, nil }

//func (i *genericIterator) Iterator() Step { return i.iter }
//func (i *genericIterator) First() Step { return i.iter }

// Next calls More and Next functions on iterator handler.
//
// If iterator step (iter field) implements ResultEvaluator it calls
// EvalResults on it before returning it. If iterator step does not implement it,
// results are omitted.
//func (i *genericIterator) nextIteration(ctx context.Context, scope *expr.Vars) (s Step, out *expr.Vars, err error) {
//	var (
//		more    bool
//		results *expr.Vars
//	)
//	if more, err = i.h.More(ctx, scope); err != nil || !more {
//		return
//	}
//
//	if results, err = i.h.Next(ctx, scope); err != nil {
//		return
//	}
//
//	if re, is := i.iter.(ResultEvaluator); is {
//		if out, err = re.EvalResults(ctx, results); err != nil {
//			return
//		}
//	}
//
//	return i.current, out, nil
//}

func GenericResourceNextCheck(useLimit bool, ptr, buffSize, total, limit uint, hasMore bool) bool {
	// if we can go more (inverted)...
	if useLimit && ptr+total >= limit {
		return false
	}

	// if we have some buffer left...
	if ptr < buffSize {
		return true
	}

	// if we can get more...
	return hasMore
}
