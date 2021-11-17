package workflows

import (
	"context"
	"testing"

	autTypes "github.com/cortezaproject/corteza-server/automation/types"
	"github.com/cortezaproject/corteza-server/pkg/logger"
	"github.com/stretchr/testify/require"
)

func Test0015_sub_workflow(t *testing.T) {
	var (
		ctx = logger.ContextWithValue(bypassRBAC(context.Background()), logger.MakeDebugLogger())
		req = require.New(t)
	)

	loadScenario(ctx, t)

	var (
		_, trace = mustExecWorkflow(ctx, t, "main", autTypes.WorkflowExecParams{Trace: true})
	)

	println(trace.String())
	req.Len(trace, 3)

	_ = req
}
