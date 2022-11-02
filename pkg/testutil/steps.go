package testutil

import (
	"testing"
)

// Step is used to build a test
type Step struct {
	Name string
	Test func(t *testing.T)
}

// Steps
type Steps []Step

// WithSteps appends Steps
func (ss Steps) WithStep(s Step) Steps {
	return append(ss, s)
}

// Run execs the various Steps that describe Tests
func (ss Steps) Run(t *testing.T) {
	for _, s := range ss {
		if !t.Run(s.Name, s.Test) {
			t.FailNow()
		}
	}
}
