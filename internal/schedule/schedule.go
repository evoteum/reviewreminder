// Package schedule validates cron expressions and reconciles reviewreminder's
// automatic-run schedule against configuration.
package schedule

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ErrExpressionRequired is returned when scheduling is enabled but no cron
// expression is configured.
var ErrExpressionRequired = errors.New("a schedule expression is required")

// Scheduler installs, reads and removes the managed schedule.
type Scheduler interface {
	// Current returns the currently scheduled expression, or "" if none.
	Current() (string, error)
	// Ensure installs or updates the schedule to expr.
	Ensure(expr string) error
	// Remove clears any managed schedule.
	Remove() error
}

// Reconcile brings s in line with the desired configuration. An invalid
// expression is refused without touching any existing schedule.
func Reconcile(s Scheduler, enabled bool, expr string) error {
	if !enabled {
		return s.Remove()
	}
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return ErrExpressionRequired
	}
	if err := Validate(expr); err != nil {
		return err
	}
	return s.Ensure(expr)
}

type fieldSpec struct{ min, max int }

// Standard five-field cron: minute, hour, day-of-month, month, day-of-week.
var fieldSpecs = []fieldSpec{{0, 59}, {0, 23}, {1, 31}, {1, 12}, {0, 7}}

// Validate checks that expr is a well-formed five-field cron expression.
func Validate(expr string) error {
	f := strings.Fields(strings.TrimSpace(expr))
	if len(f) != 5 {
		return fmt.Errorf("cron expression %q must have 5 fields, got %d", expr, len(f))
	}
	for i, field := range f {
		if err := validateField(field, fieldSpecs[i]); err != nil {
			return fmt.Errorf("cron expression %q: %w", expr, err)
		}
	}
	return nil
}

func validateField(field string, spec fieldSpec) error {
	for _, part := range strings.Split(field, ",") {
		if err := validatePart(part, spec); err != nil {
			return err
		}
	}
	return nil
}

func validatePart(part string, spec fieldSpec) error {
	base := part
	if i := strings.Index(part, "/"); i >= 0 {
		step := part[i+1:]
		base = part[:i]
		if n, err := strconv.Atoi(step); err != nil || n <= 0 {
			return fmt.Errorf("invalid step in %q", part)
		}
	}
	if base == "*" {
		return nil
	}
	if i := strings.Index(base, "-"); i >= 0 {
		a, errA := strconv.Atoi(base[:i])
		b, errB := strconv.Atoi(base[i+1:])
		if errA != nil || errB != nil {
			return fmt.Errorf("invalid range in %q", part)
		}
		if a < spec.min || b > spec.max || a > b {
			return fmt.Errorf("range out of bounds in %q", part)
		}
		return nil
	}
	v, err := strconv.Atoi(base)
	if err != nil {
		return fmt.Errorf("invalid value %q", part)
	}
	if v < spec.min || v > spec.max {
		return fmt.Errorf("value %d out of bounds in %q", v, part)
	}
	return nil
}
