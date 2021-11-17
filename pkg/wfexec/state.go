package wfexec

import (
	"encoding/json"
	"time"

	"github.com/cortezaproject/corteza-server/pkg/auth"
	"github.com/cortezaproject/corteza-server/pkg/expr"
)

type (
	// State holds information about Session ID
	State struct {
		// state identifier
		stateId uint64

		// parent state
		// root states have null
		parent *State

		created   time.Time
		completed *time.Time

		// who's running this?
		owner auth.Identifiable

		// Session identifier
		session *Session

		// previous step
		pStep Step

		// current step
		cStep Step

		// next steps
		//next Steps

		// step error (if any)
		//err error

		// input variables that were sent to resume the session
		input *expr.Vars

		// scope
		scope *expr.Vars

		// step execution results
		results *expr.Vars

		// error handling step
		//errHandler Step

		// error handled flag, this gets restarted on every new state!
		//errHandled bool

		//loops []Iterator

		traverser block

		action string

		//finalize func(*expr.Vars)
	}
)

func RootState(ses *Session, owner auth.Identifiable, step Step, scope *expr.Vars) *State {
	return &State{
		stateId:   nextID(),
		created:   *(now()),
		owner:     owner,
		session:   ses,
		cStep:     step,
		scope:     scope,
		traverser: &Block{g: ses.g},
	}
}

func (s *State) next(steps ...Step) (states []*State) {
	states = make([]*State, len(steps))

	if len(steps) == 1 {
		// one outbound path,
		// reuse state
		s.pStep = s.cStep
		s.cStep = steps[0]
		states[0] = s
		return
	}

	for i, step := range steps {
		states[i] = &State{
			parent:    s,
			stateId:   nextID(),
			created:   *(now()),
			owner:     s.owner,
			session:   s.session,
			scope:     s.scope,
			cStep:     step,
			pStep:     s.cStep,
			traverser: &Block{g: s.session.g},
		}
	}

	return
}

func (s *State) withTraverser(t block) *State {
	if t == nil {
		panic("traversal can not be nil")
	}

	return &State{
		stateId:   nextID(),
		created:   *(now()),
		parent:    s,
		owner:     s.owner,
		session:   s.session,
		scope:     s.scope,
		pStep:     s.pStep,
		cStep:     s.cStep,
		traverser: t,
	}
}

func (s State) isRoot() bool {
	return s.parent == nil
}

func (s *State) level() (l int) {
	p := s
	for {
		if p.isRoot() {
			return
		}

		p = p.parent
		l++
	}
}

func (s *State) exit() (*State, *expr.Vars) {
	if s.isRoot() {
		panic("can not pop root state")
	}
	s.completed = now()

	return s.parent, s.scope
}

// states without traverser are considered final
//
// We're keeping the
func (s *State) terminate() *State {
	s.traverser = nil
	s.completed = now()
	return s
}

//func NewState(ses *Session, parent *State, owner auth.Identifiable, caller, current Step, scope *expr.Vars) *State {
//	return &State{
//		parent:    parent,
//		stateId:   nextID(),
//		owner:     owner,
//		sessionId: ses.id,
//		created:   *now(),
//		step:      current,
//		scope:     scope,
//
//		//loops:      make([]Iterator, 0, 4),
//		graph: func() block {
//			if parent == nil {
//				return &Block{g: ses.g}
//			}
//
//			return parent.g
//		}(),
//	}
//}

//func FinalState(ses *Session, scope *expr.Vars) *State {
//	return &State{
//		stateId:   nextID(),
//		sessionId: ses.id,
//		created:   *now(),
//		completed: now(),
//		scope:     scope,
//	}
//}

//func (s State) Next(current Step, scope *expr.Vars) *State {
//	return &State{
//		stateId:    nextID(),
//		owner:      s.owner,
//		sessionId:  s.sessionId,
//		parent:     s.step,
//		errHandler: s.errHandler,
//		blockStack: s.blockStack,
//
//		step:  current,
//		scope: scope,
//	}
//}

func (s State) MakeRequest() *ExecRequest {
	return &ExecRequest{
		SessionID: s.session.ID(),
		StateID:   s.stateId,
		Scope:     s.scope,
		Input:     s.input,
		Parent:    s.pStep,
	}
}

//func (s *State) newLoop(i Iterator) {
//	s.loops = append(s.loops, i)
//}

//func (s *State) enterBlock(b block) {
//	s.blockStack = append(s.blockStack, b)
//}
//
//func (s *State) exitBlock() (Steps, *expr.Vars) {
//	top := s.blockStack.top()
//	if top == nil {
//		// root block
//		return nil, s.scope
//	}
//
//	s.blockStack = s.blockStack[:len(s.blockStack)-1]
//	return top.Exit()
//}

// handle error by finding a state with error handling traversal
func (s *State) handleError(err error) (Step, error) {
	p := s
	for {
		if p.isRoot() {
			// root state can not be an error handler
			return nil, err
		}

		if eh, is := p.traverser.(*errHandler); is {
			// Error handler found, expecting error
			// to be handled
			return eh.HandleError(err)
		}

		p = p.parent
	}
}

//// ends loop and returns step that leads out of the loop
//func (s *State) loopEnd() (out Steps) {
//	l := len(s.loops) - 1
//	if l < 0 {
//		panic("not inside a loop")
//	}
//
//	out = Steps{s.loops[l].Break()}
//	s.loops = s.loops[:l]
//	return
//}
//
//func (s State) loopCurr() Iterator {
//	l := len(s.loops)
//	if l > 0 {
//		return s.loops[l-1]
//	}
//
//	return nil
//}

func (s State) MakeFrame() *Frame {
	var (
		// might not be the most optimal way but we need to
		// un-reference scope, input, result variables
		unref = func(vars *expr.Vars) *expr.Vars {
			out := &expr.Vars{}
			tmp, _ := json.Marshal(vars)
			_ = json.Unmarshal(tmp, out)
			return out
		}
	)

	f := &Frame{
		CreatedAt: s.created,
		SessionID: s.session.id,
		StateID:   s.stateId,
		Input:     unref(s.input),
		Scope:     unref(s.scope),
		Results:   unref(s.results),
		//NextSteps: s.next.IDs(),
		Action: s.action,
	}

	//if s.err != nil {
	//	f.Error = s.err.Error()
	//}

	if s.cStep != nil {
		f.StepID = s.cStep.ID()
	}

	if s.pStep != nil {
		f.ParentID = s.pStep.ID()
	}

	if s.completed != nil {
		f.StepTime = uint(s.completed.Sub(s.created) / time.Millisecond)
	}

	return f
}

//func (s *State) Error() string {
//	//if s.err == nil {
//	//	return ""
//	//}
//	//
//	//return s.err.Error()
//	return ""
//}
