// Package perf provides facilities to measure and report performance
package perf

import (
	"time"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/utils/logging"
)

// Measure tracks the operation to be measured and returns a function that can be
// run in a defer block of the caller to output the time taken for the operation.
func Measure(s string) func() {
	if logging.PerfLogger == nil {
		return func() {}
	}
	start := time.Now()
	return func() {
		logging.PerfLogger.Printf("%s [%s]", s, time.Since(start))
	}
}
