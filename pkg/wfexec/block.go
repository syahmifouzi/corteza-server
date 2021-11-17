package wfexec

import (
	"context"

	"github.com/cortezaproject/corteza-server/pkg/expr"
)

type (
	block interface {
		//Init() error
		//CurrentStep() Step
		//FirstStep() Step

		// Next returns zero, one or more steps that
		// should be executed in the following session-exec cycle
		Next(context.Context, Step) (Steps, error)

		// Is the current pos
		//Valid() bool

		Children(Step) Steps

		//Exec(ctx context.Context, r *ExecRequest) (ExecResponse, error)
		//Exit() (Steps, *expr.Vars)
		//IsLoop() bool
	}

	iterator interface {
		block
		Continue() Step
	}

	blockStack []block

	Block struct {
		g *Graph

		current Step
		next    Steps

		// @todo Should stack be kept on the block (and not on the state)?
		scope *expr.Vars
	}
)

var _ block = &Block{}

//
//// exists from the topmost block
//// and modifies the block stack
//func (bb *blockStack) exit() (Steps, *expr.Vars) {
//	l := len(*bb) - 1
//	if l < 0 {
//		panic("not inside a loop")
//	}
//
//	top := (*bb)[l]
//	*bb = (*bb)[:l]
//	return top.Exit()
//}

// returns top block from the block stack
func (bb blockStack) top() block {
	l := len(bb)
	if l == 0 {
		return nil
	}

	return bb[l-1]
}

// returns top block from the block stack
func (bb blockStack) root() bool {
	return len(bb) == 1
}

func (b *Block) Next(_ context.Context, c Step) (Steps, error) {
	return b.g.Children(c), nil
}

func (b *Block) Children(c Step) Steps {
	return b.g.Children(c)
}

//func (b *Block) Exit() (Steps, *expr.Vars) {
//	return nil, nil
//}
