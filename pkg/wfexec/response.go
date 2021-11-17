package wfexec

import (
	"time"
)

type (
	ExecResponse interface{}
	partial      struct{}

	termination struct{}

	delayed struct {
		// when not nil, assuming delayed
		resumeAt time.Time

		// state to be resumed
		state *State
	}

	// when session is resumed from a delay we'll replace
	// delay step on state with the a generic step that will return resumed{}
	resumed struct{}
)

func Delay(until time.Time) *delayed {
	return &delayed{resumeAt: until}
}

func Resume() *resumed {
	return &resumed{}
}

func Termination() *termination {
	return &termination{}
}

type (
	exitBlock struct{}
	//loopBreak    struct{}
	loopContinue struct{}
)

func ExitBlock() *exitBlock       { return &exitBlock{} }
func LoopContinue() *loopContinue { return &loopContinue{} }
