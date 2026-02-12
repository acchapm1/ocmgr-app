// Package resolver walks the profile "extends" dependency chain and
// returns a flattened, de-duplicated list of profile names in apply
// order (parents first, children last).
package resolver

import (
	"fmt"
	"strings"
)

// Loader retrieves the extends field for a given profile name.
// It is typically backed by store.Get(name).Extends.
type Loader func(name string) (extends string, err error)

// Resolve expands the requested profile names by walking each
// profile's extends chain.  The returned slice is ordered so that
// parent profiles appear before their children and no name appears
// more than once.
//
// A circular dependency (e.g. a → b → a) is detected and reported
// as an error.
func Resolve(names []string, load Loader) ([]string, error) {
	// seen tracks profiles already placed in the result list so we
	// never add a duplicate.
	seen := make(map[string]bool)

	var result []string

	for _, name := range names {
		chain, err := walkChain(name, load)
		if err != nil {
			return nil, err
		}
		// chain is ordered [root ancestor … name].
		for _, n := range chain {
			if !seen[n] {
				seen[n] = true
				result = append(result, n)
			}
		}
	}

	return result, nil
}

// walkChain builds the full dependency chain for a single profile,
// from the root ancestor down to the given name.
//
// It returns an error if a circular dependency is detected or the
// loader fails.
func walkChain(name string, load Loader) ([]string, error) {
	// visiting tracks the current chain to detect cycles.
	visiting := make(map[string]bool)
	var chain []string

	current := name
	for current != "" {
		if visiting[current] {
			return nil, fmt.Errorf("circular dependency detected: %s", formatCycle(chain, current))
		}
		visiting[current] = true
		chain = append(chain, current)

		extends, err := load(current)
		if err != nil {
			return nil, fmt.Errorf("resolving profile %q: %w", current, err)
		}
		current = strings.TrimSpace(extends)
	}

	// chain is [name, parent, grandparent, …].
	// Reverse it so the root ancestor comes first.
	reverse(chain)
	return chain, nil
}

// reverse reverses a string slice in place.
func reverse(s []string) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

// formatCycle produces a human-readable cycle description like
// "a → b → c → a".
func formatCycle(chain []string, loopBack string) string {
	parts := make([]string, len(chain)+1)
	copy(parts, chain)
	parts[len(chain)] = loopBack
	return strings.Join(parts, " → ")
}
