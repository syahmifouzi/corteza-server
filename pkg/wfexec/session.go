package wfexec

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/cortezaproject/corteza-server/pkg/auth"
	"github.com/cortezaproject/corteza-server/pkg/expr"
	"github.com/cortezaproject/corteza-server/pkg/id"
	"github.com/cortezaproject/corteza-server/pkg/logger"
	"go.uber.org/zap"
)

type (
	Session struct {
		// Session identifier
		id uint64

		workflowID uint64

		// steps graph
		g *Graph

		started time.Time

		// state channel (ie work queue)
		qState chan *State

		// crash channel
		qErr chan error

		// locks concurrent executions
		execLock chan struct{}

		// delayed states (waiting for the right time)
		delayed map[uint64]*delayed

		// prompted
		prompted map[uint64]*prompted

		// how often we check for delayed states and how often idle stat is checked in Wait()
		workerInterval time.Duration

		// only one worker routine per session
		workerLock chan struct{}

		statusChange chan int

		// holds final result
		result *expr.Vars
		err    error

		mux sync.RWMutex

		// debug logger
		log *zap.Logger

		dumpStacktraceOnPanic bool

		eventHandler StateChangeHandler

		callStack []uint64
	}

	StateChangeHandler func(SessionStatus, *State, *Session)

	SessionOpt func(*Session)

	Frame struct {
		CreatedAt time.Time  `json:"createdAt"`
		SessionID uint64     `json:"sessionID"`
		StateID   uint64     `json:"stateID"`
		Input     *expr.Vars `json:"input"`
		Scope     *expr.Vars `json:"scope"`
		Results   *expr.Vars `json:"results"`
		ParentID  uint64     `json:"parentID"`
		StepID    uint64     `json:"stepID"`
		NextSteps []uint64   `json:"nextSteps"`

		// How much time from the 1st step to the start of this step in milliseconds
		ElapsedTime uint `json:"elapsedTime"`

		// How much time it took to execute this step in milliseconds
		StepTime uint `json:"stepTime"`

		Action string `json:"action,omitempty"`
		Error  string `json:"error,omitempty"`
	}

	// ExecRequest is passed to Exec() functions and contains all information
	// for step execution
	ExecRequest struct {
		SessionID uint64
		StateID   uint64

		// Current input received on session resume
		Input *expr.Vars

		// Current scope
		Scope *expr.Vars

		// Helps with gateway join/merge steps
		// that needs info about the step it's currently merging
		Parent Step
	}

	SessionStatus int

	callStackCtxKey struct{}
)

const (
	sessionStateChanBuf   = 512
	sessionConcurrentExec = 32
)

const (
	SessionActive SessionStatus = iota
	SessionPrompted
	SessionDelayed
	SessionRecovered
	SessionFailed
	SessionCompleted
)

var (
	// wrapper around nextID that will aid service testing
	nextID = func() uint64 {
		return id.Next()
	}

	// wrapper around time.Now() that will aid service testing
	now = func() *time.Time {
		c := time.Now()
		return &c
	}
)

func (s SessionStatus) String() string {
	switch s {
	case SessionActive:
		return "active"
	case SessionPrompted:
		return "prompted"
	case SessionDelayed:
		return "delayed"
	case SessionRecovered:
		return "recovered"
	case SessionFailed:
		return "failed"
	case SessionCompleted:
		return "completed"
	}

	return "UNKNOWN-SESSION-STATUS"
}

func NewSession(ctx context.Context, g *Graph, oo ...SessionOpt) *Session {
	s := &Session{
		g:        g,
		id:       nextID(),
		started:  *now(),
		qState:   make(chan *State, sessionStateChanBuf),
		qErr:     make(chan error, 1),
		execLock: make(chan struct{}, sessionConcurrentExec),
		delayed:  make(map[uint64]*delayed),
		prompted: make(map[uint64]*prompted),

		//workerInterval: time.Millisecond,
		workerInterval: time.Millisecond * 250, // debug mode rate
		workerLock:     make(chan struct{}, 1),

		log: zap.NewNop(),

		eventHandler: func(SessionStatus, *State, *Session) {
			// noop
		},
	}

	for _, o := range oo {
		o(s)
	}

	s.log = s.log.
		With(zap.Uint64("sessionID", s.id))

	s.callStack = append(s.callStack, s.id)

	go s.worker(ctx)

	return s
}

func (s *Session) Status() SessionStatus {
	s.mux.RLock()
	defer s.mux.RUnlock()

	switch {
	case s.err != nil:
		return SessionFailed

	case len(s.prompted) > 0:
		return SessionPrompted

	case len(s.delayed) > 0:
		return SessionDelayed

	case s.result == nil:
		return SessionActive

	default:
		return SessionCompleted
	}
}

func (s *Session) ID() uint64 {
	return s.id
}

func (s *Session) Idle() bool {
	return s.Status() != SessionActive
}

func (s *Session) Error() error {
	s.mux.RLock()
	defer s.mux.RUnlock()

	return s.err
}

func (s *Session) Result() *expr.Vars {
	s.mux.RLock()
	defer s.mux.RUnlock()

	return s.result
}

func (s *Session) Exec(ctx context.Context, step Step, scope *expr.Vars) error {
	s.mux.RLock()
	defer s.mux.RUnlock()

	err := func() error {
		if s.g.Len() == 0 {
			return fmt.Errorf("refusing to execute without steps")
		}

		if len(s.g.Parents(step)) > 0 {
			return fmt.Errorf("cannot execute step with parents")
		}
		return nil
	}()

	if err != nil {
		// send nil to error queue to trigger worker shutdown
		// session error must be set to update session status
		s.qErr <- err
		return err
	}

	if scope == nil {
		scope, _ = expr.NewVars(nil)
	}

	return s.enqueue(ctx, RootState(s, auth.GetIdentityFromContext(ctx), step, scope))
}

// UserPendingPrompts prompts fn returns all owner's pending prompts on this session
func (s *Session) UserPendingPrompts(ownerId uint64) (out []*PendingPrompt) {
	if ownerId == 0 {
		return
	}

	defer s.mux.RUnlock()
	s.mux.RLock()

	out = make([]*PendingPrompt, 0, len(s.prompted))

	for _, p := range s.prompted {
		if p.ownerId != ownerId {
			continue
		}

		pending := p.toPending()
		pending.SessionID = s.id
		out = append(out, pending)
	}

	return
}

// AllPendingPrompts returns all pending prompts for all user
func (s *Session) AllPendingPrompts() (out []*PendingPrompt) {
	defer s.mux.RUnlock()
	s.mux.RLock()

	out = make([]*PendingPrompt, 0, len(s.prompted))

	for _, p := range s.prompted {
		pending := p.toPending()
		pending.SessionID = s.id
		out = append(out, pending)
	}

	return
}

func (s *Session) Resume(ctx context.Context, stateId uint64, input *expr.Vars) (*ResumedPrompt, error) {
	defer s.mux.Unlock()
	s.mux.Lock()

	var (
		i      = auth.GetIdentityFromContext(ctx)
		p, has = s.prompted[stateId]
	)
	if !has {
		return nil, fmt.Errorf("unexisting state")
	}

	if i == nil || p.ownerId != i.Identity() {
		return nil, fmt.Errorf("state access denied")
	}

	delete(s.prompted, stateId)

	// setting received input to state
	p.state.input = input

	if err := s.enqueue(ctx, p.state); err != nil {
		return nil, err
	}

	return p.toResumed(), nil
}

func (s *Session) canEnqueue(st *State) error {
	if st == nil {
		return fmt.Errorf("state is nil")
	}

	// when the step is completed right away, it is considered as special
	if st.cStep == nil && st.completed == nil {
		return fmt.Errorf("state step is nil")
	}

	return nil
}

func (s *Session) enqueue(ctx context.Context, st *State) error {
	if err := s.canEnqueue(st); err != nil {
		return err
	}

	if st.stateId == 0 {
		st.stateId = nextID()
	}

	select {
	case <-ctx.Done():
		return ctx.Err()

	case s.qState <- st:
		return nil
	}
}

// Wait does not wait for the whole wf to be complete but until:
//  - context timeout
//  - idle state
//  - error in error queue
func (s *Session) Wait(ctx context.Context) error {
	return s.WaitUntil(ctx, SessionFailed, SessionDelayed, SessionCompleted)
}

// WaitUntil blocks until workflow session gets into expected status
//
func (s *Session) WaitUntil(ctx context.Context, expected ...SessionStatus) error {
	indexed := make(map[SessionStatus]bool)
	for _, status := range expected {
		indexed[status] = true
	}

	// already at the expected status
	if indexed[s.Status()] {
		return s.err
	}

	s.log.Debug(
		"waiting for status change",
		zap.Any("expecting", expected),
		zap.Duration("interval", s.workerInterval),
	)

	waitCheck := time.NewTicker(s.workerInterval)
	defer waitCheck.Stop()

	for {
		select {
		case <-waitCheck.C:
			if indexed[s.Status()] {
				s.log.Debug("waiting complete", zap.Stringer("status", s.Status()))
				// nothing in the pipeline
				return s.err
			}

		case <-ctx.Done():
			s.log.Debug("wait context done", zap.Error(ctx.Err()))
			s.Cancel()
			return s.err
		}
	}
}

func (s *Session) worker(ctx context.Context) {
	defer s.Stop()

	// making sure
	defer close(s.workerLock)
	s.workerLock <- struct{}{}

	workerTicker := time.NewTicker(s.workerInterval)

	defer workerTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.log.Debug("worker context done", zap.Error(ctx.Err()))
			return

		case <-workerTicker.C:
			s.queueScheduledSuspended()

		case st := <-s.qState:
			if st == nil {
				// stop worker
				s.log.Debug("completed")
				return
			}

			if st.level() > 10 {
				panic("too low")
			}

			log := s.log.With(
				zap.Uint64("stateID", st.stateId),
				zap.Uint64("stepId", st.cStep.ID()),
				zap.Int("depth", st.level()),
			)

			log.Debug("pulled state from queue")

			if st.traverser == nil {
				log.Debug("done, setting results and stopping the worker")

				func() {
					// mini lambda fn to ensure we can properly unlock with defer
					s.mux.Lock()
					defer s.mux.Unlock()

					// with merge we are making sure
					// result != nil even if state scope is
					s.result = (&expr.Vars{}).MustMerge(st.scope)
				}()

				// Call event handler with completed status
				s.eventHandler(SessionCompleted, st, s)

				return
			}

			// add empty struct to chan to lock and to have control over number of concurrent go processes
			// this will block if number of items in execLock chan reached value of sessionConcurrentExec
			s.execLock <- struct{}{}

			go func() {
				defer func() {
					// remove protection that prevents multiple
					// steps executing at the same time
					<-s.execLock
				}()

				nxt, err := s.exec(ctx, log, st)

				if err != nil {
					log.Debug("is there error to be handled", zap.Error(err))
					st.pStep = st.cStep
					st.cStep, err = st.handleError(err)
					log.Debug("is the error handled", zap.Error(err), zap.Any("ehStep", st.cStep))
					if err != nil {
						// wrap the error with workflow and step information
						err = fmt.Errorf(
							"workflow %d step %d execution failed: %w",
							s.workflowID,
							st.cStep.ID(),
							err,
						)

						// We need to force failed session status
						// because it's not set early enough to pick it up with s.Status()
						s.eventHandler(SessionFailed, st, s)

						// pushing step execution error into error queue
						// to break worker loop
						s.qErr <- err
						return
					}

					// error handed, session recovered
					s.eventHandler(SessionRecovered, st, s)

					// alter the next states to be handled (since there are no
					// states from the nxt anymore...
					nxt = []*State{st}

					log.Debug("next state", zap.Any("step", st.cStep))
				}

				s.eventHandler(s.Status(), st, s)

				log.Debug("next states?", zap.Int("count", len(nxt)), zap.Any("list", nxt))
				for _, n := range nxt {
					if err = s.enqueue(ctx, n); err != nil {
						log.Error("unable to enqueue", zap.Error(err))
						return
					}
				}
			}()

		case err := <-s.qErr:
			s.mux.Lock()
			defer s.mux.Unlock()

			if err == nil {
				// stop worker
				return
			}

			// set final error on session
			s.err = err
			return
		}
	}
}

func (s *Session) Cancel() {
	s.log.Debug("canceling")
	s.qErr <- fmt.Errorf("canceled")
}

func (s *Session) Stop() {
	s.log.Debug("stopping worker")
	s.qErr <- nil
}

func (s *Session) Suspended() bool {
	defer s.mux.RUnlock()
	s.mux.RLock()
	return len(s.delayed) > 0
}

func (s *Session) queueScheduledSuspended() {
	defer s.mux.Unlock()
	s.mux.Lock()

	for id, sus := range s.delayed {
		if !sus.resumeAt.IsZero() && sus.resumeAt.After(*now()) {
			continue
		}

		delete(s.delayed, id)

		// Set state input when step is resumed
		sus.state.input = &expr.Vars{}
		sus.state.input.Set("resumed", true)
		sus.state.input.Set("resumeAt", sus.resumeAt)
		s.qState <- sus.state
	}
}

// executes single step, resolves response and schedule following steps for execution
func (s *Session) exec(ctx context.Context, log *zap.Logger, st *State) (nxt []*State, err error) {
	//if st.input != nil {
	//	spew.Dump("st.input:", st.input.Dict())
	//}
	//if st.results != nil {
	//	spew.Dump("st.results:", st.results.Dict())
	//}
	//if st.scope != nil {
	//	spew.Dump("st.scope:", st.scope.Dict())
	//}

	var (
		next   Steps
		result ExecResponse

		// what's the current block?
		currentBlock = st.traverser
		// @todo remove currLoop = st.loopCurr()

		scope = (&expr.Vars{}).MustMerge(st.scope)
	)

	defer func() {
		reason := recover()
		if reason == nil {
			return
		}

		var (
			perr error
			loc  = identifyPanic()
		)

		// normalize error and set it to state
		switch reason := reason.(type) {
		case error:
			perr = fmt.Errorf("step %d crashed in %s: %w", st.cStep.ID(), loc, reason)
		default:
			perr = fmt.Errorf("step %d crashed in %s: %v", st.cStep.ID(), loc, reason)
		}

		if s.dumpStacktraceOnPanic {
			fmt.Printf("Error: %v\n", perr)
		}

		s.qErr <- perr
	}()

	//if st.step != nil {
	log = log.With(zap.Uint64("stepID", st.cStep.ID()))
	//}

	{

		//if inBlock != nil && inBlock.Initiator(st.step) {
		//	result = inBlock
		//	// @todo remove } else if currLoop != nil && currLoop.Is(st.step) {
		//	// @todo remove 	result = currLoop
		//} else {
		// push logger to context but raise the stacktrace level to panic
		// to prevent overly verbose traces
		ctx = logger.ContextWithValue(ctx, log)

		// Context received in exec() wil not have the identity we're expecting
		// we need to pull it from state owner and add it to new context
		// that is set to step exec function
		stepCtx := auth.SetIdentityToContext(ctx, st.owner)
		stepCtx = SetContextCallStack(stepCtx, s.callStack)

		//result, st.err = st.cStep.Exec(stepCtx, st.MakeRequest())
		if result, err = st.cStep.Exec(stepCtx, st.MakeRequest()); err != nil {
			return nil, err
		}

		//if iterator, isIterator := result.(Iterator); isIterator && st.err == nil {
		//	// Exec fn returned an iterator, adding loop to stack
		//	st.newLoop(iterator)
		//	if err = iterator.Start(ctx, scope); err != nil {
		//		return
		//	}
		//}
		//if sub, is := result.(block); is {
		//	st.blockStack = append(st.blockStack, sub)
		//	currentBlock = sub
		//}
		//}

		//if st.err != nil {
		//	if st.errHandler == nil {
		//		// no error handler set
		//		return nil, st.err
		//	}
		//
		//	// handling error with error handling
		//	// step set in one of the previous steps
		//	log.Warn("step execution error handled",
		//		zap.Uint64("errorHandlerStepId", st.errHandler.ID()),
		//		zap.Error(st.err),
		//	)
		//
		//	_ = expr.Assign(scope, "error", expr.Must(expr.NewString(st.err.Error())))
		//
		//	// copy error handler & disable it on state to prevent inf. loop
		//	// in case of another error in the error-handling branch
		//	eh := st.errHandler
		//	st.errHandler = nil
		//	st.errHandled = true
		//	return []*State{st.Next(eh, scope)}, nil
		//}

		// respect step's request to go into a new block
		//switch b := result.(type) {
		//case block:
		//	//st.action = "new block on stack"
		//	// add looper to state
		//	//var (
		//	//	n Step
		//	//)
		//	if st.err = b.Init(); st.err != nil {
		//		return nil, st.err
		//	}
		//
		//	// complete this exec and start with the substate & subblock
		//	return []*State{st.Block(b, scope)}, nil
		//	//b.FirstStep()
		//	//
		//	//n, result, st.err = b.FirstStep().Exec(ctx, scope)
		//	//if st.err != nil {
		//	//	return nil, st.err
		//	//}
		//	//
		//	//if n == nil {
		//	//	st.next = st.loopEnd()
		//	//} else {
		//	//	st.next = Steps{n}
		//	//}
		//	//case Iterator:
		//	//	st.action = "iterator initialized"
		//	//	// add looper to state
		//	//	var (
		//	//		n Step
		//	//	)
		//	//	n, result, st.err = l.Next(ctx, scope)
		//	//	if st.err != nil {
		//	//		return nil, st.err
		//	//	}
		//	//
		//	//	if n == nil {
		//	//		st.next = st.loopEnd()
		//	//	} else {
		//	//		st.next = Steps{n}
		//	//	}
		//}

		log.Debug(
			"step executed",
			zap.String("resultType", fmt.Sprintf("%T", result)),
		)
		switch result := result.(type) {
		//case *errHandler:
		//	st.action = "error handler initialized"
		//	// this step sets error handling step on current state
		//	// and continues on the current path
		//	//st.errHandler = result.handler
		//	st.enterBlock(result)
		//	st.next, err = result.Next(ctx, st.step)
		//	if err != nil {
		//		return
		//	}
		//
		//	return []*State{st}, nil
		//
		//	// find step that's not error handler and
		//	// use it for the next step
		//	// @todo rework this and do error-handling through "block"
		//	//for _, c := range currentBlock.NextStep() {
		//	//	if c != st.errHandler {
		//	//		st.next = Steps{c}
		//	//		break
		//	//	}
		//	//}
		//	//panic("error handling is pending reimplementation")

		case block:
			// Push state
			log.Debug("new block")
			st = st.withTraverser(result)

		case *expr.Vars:
			// most common (successful) result
			// session will continue with configured child steps
			st.results = result
			scope = scope.MustMerge(st.results)

		case *exitBlock:
			st.action = "block break"

			// jump out of the loop
			st, st.results = st.exit()
			log.Debug("breaking from block")

		case *loopContinue:
			if itr, is := currentBlock.(iterator); !is {
				return nil, fmt.Errorf("block is not an iterator")
			} else {
				st.action = "loop continue"

				// jump back to iterator step
				next = Steps{itr.Continue()}
				log.Debug("continuing with next iteration")
			}

		case *partial:
			st.action = "partial"
			// *partial is returned when step needs to be executed again
			// it's used mainly for join gateway step that should be called multiple times (one for each parent path)
			return

		case *termination:
			st.action = "termination"
			// terminate all activities, all delayed tasks and exit right away
			log.Debug("termination", zap.Int("delayed", len(s.delayed)))
			s.mux.Lock()
			s.delayed = nil

			// @todo -- should remove all prompts as well?
			//s.prompted = nil
			s.mux.Unlock()

			st.terminate()
			return []*State{st}, nil

		case *delayed:
			st.action = "delayed"
			log.Debug("session delayed", zap.Time("at", result.resumeAt))

			result.state = st
			s.mux.Lock()
			s.delayed[st.stateId] = result
			s.mux.Unlock()

			// execution will be resumed after the delay
			return

		case *resumed:
			st.action = "resumed"
			log.Debug("session resumed")

		case *prompted:
			st.action = "prompted"
			if result.ownerId == 0 {
				return nil, fmt.Errorf("prompt without an owner")
			}

			result.state = st
			s.mux.Lock()
			s.prompted[st.stateId] = result
			s.mux.Unlock()

			// execution will be resumed after the prompt response
			return

		case Steps:
			st.action = "next-steps"
			// session continues with set of specified steps
			// steps MUST be configured in a graph as step's children
			next = result

		case Step:
			st.action = "next-step"
			// session continues with a specified step
			// step MUST be configured in a graph as step's child
			next = Steps{result}

		default:
			return nil, fmt.Errorf("unknown exec response type %T", result)
		}
	}

	//if len(next) == 0 {
	//	// step's exec did not return next steps (only gateway steps, iterators and loops controls usually do that)
	//	//
	//	// rely on graph and get next (children) steps from there
	//	if next, err = currentBlock.Next(ctx, st.cStep); err != nil {
	//		return nil, err
	//	}
	//}

	log.Debug("scope", zap.Any("scope", scope.Dict()))
	st.scope = scope

	if len(next) == 0 {
		next, err = st.traverser.Next(ctx, st.cStep)
		log.Debug("next from traversal", zap.Any("step", next), zap.Error(err))
		if err != nil {
			return nil, err
		}
	} else {
		log.Debug("checking next steps", zap.Any("steps", next), zap.Error(err))
		// children returned from step's exec
		// do a quick sanity check
		cc := st.traverser.Children(st.cStep)
		if len(cc) > 0 && !cc.Contains(next...) {
			return nil, fmt.Errorf("inconsistent relationship")
		}
	}

	//if len(st.next) == 0 && currentBlock.IsLoop() {
	//	// gracefully handling last step of iteration branch
	//	// that does not point back to the iterator step
	//	st.next = Steps{currentBlock.FirstStep()}
	//	log.Debug("last step in iteration branch, going back", zap.Uint64("backStepId", st.next[0].ID()))
	//}

	if len(next) == 0 {
		log.Debug("zero paths, finalizing", zap.Any("scope", scope.Dict()))
		// using state to transport results and complete the worker loop
		st.terminate()
		return []*State{st}, nil
	}

	log.Debug("forking", zap.Int("children", len(next)), zap.Any("scope", scope.Dict()))
	return st.next(next...), nil

	//for _, st = range st.fork(next...) {
	//	if err = s.canEnqueue(st); err != nil {
	//		log.Error("unable to queue", zap.Error(err))
	//		return
	//	}
	//
	//	nxt[i] = nn
	//
	//}
	//
	//nxt = make([]*State, len(next))
	//for i, step := range next {
	//}
	//
	//return nxt, nil
}

func SetWorkerInterval(i time.Duration) SessionOpt {
	return func(s *Session) {
		s.workerInterval = i
	}
}

func SetHandler(fn StateChangeHandler) SessionOpt {
	return func(s *Session) {
		s.eventHandler = fn
	}
}

func SetWorkflowID(workflowID uint64) SessionOpt {
	return func(s *Session) {
		s.workflowID = workflowID
	}
}

func SetLogger(log *zap.Logger) SessionOpt {
	return func(s *Session) {
		s.log = log
	}
}

func SetDumpStacktraceOnPanic(dump bool) SessionOpt {
	return func(s *Session) {
		s.dumpStacktraceOnPanic = dump
	}
}

func SetCallStack(id ...uint64) SessionOpt {
	return func(s *Session) {
		s.callStack = id
	}
}

func SetContextCallStack(ctx context.Context, ss []uint64) context.Context {
	return context.WithValue(ctx, callStackCtxKey{}, ss)
}

func GetContextCallStack(ctx context.Context) []uint64 {
	v := ctx.Value(callStackCtxKey{})
	if v == nil {
		return nil
	}

	return v.([]uint64)
}

func identifyPanic() string {
	var name, file string
	var line int
	var pc [16]uintptr

	n := runtime.Callers(3, pc[:])
	for _, pc := range pc[:n] {
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		file, line = fn.FileLine(pc)
		name = fn.Name()
		if !strings.HasPrefix(name, "runtime.") {
			break
		}
	}

	switch {
	case name != "":
		return fmt.Sprintf("%v:%v", name, line)
	case file != "":
		return fmt.Sprintf("%v:%v", file, line)
	}

	return fmt.Sprintf("pc:%x", pc)
}
