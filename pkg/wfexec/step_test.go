package wfexec

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSteps(t *testing.T) {
	var (
		req = require.New(t)

		a = &sesTestStep{Name: "a"}
		b = &sesTestStep{Name: "b"}
		c = &sesTestStep{Name: "c"}
		d = &sesTestStep{Name: "d"}

		abc = Steps{a, b, c}
	)

	req.True(abc.Contains(a))
	req.False(abc.Contains(d))
	req.Equal(abc, abc.Exclude())
	req.Equal(Steps{a, c}, abc.Exclude(b))
}
