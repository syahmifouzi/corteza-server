package wfexec

import (
	"context"
)

type (
	errHandler struct {
		Block
		g            *Graph
		ehStep       Step
		handledError error
	}
)

func ErrorHandler(g *Graph, h Step) *errHandler {
	return &errHandler{g: g, ehStep: h}
}

var _ block = &errHandler{}

// Next on errHandler returns all outbound non-error-handling steps
func (eh *errHandler) Next(_ context.Context, c Step) (Steps, error) {
	println("asking err handler what to do next")
	if eh.handledError == nil {
		println("no error was handled yet, returning all children but error handler")
		return eh.g.Children(c).Exclude(eh.ehStep), nil
	} else {
		println("no error was handled, returning error handler outbound")
		return Steps{eh.ehStep}, nil
	}
}

func (eh *errHandler) HandleError(err error) (Step, error) {
	println("error handled!!! yayz", err)
	eh.handledError = err
	// @todo set error to scope
	return eh.ehStep, nil
}
