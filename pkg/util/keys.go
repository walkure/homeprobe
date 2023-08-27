package util

import (
	"cmp"
	"slices"
)

// Keys returns a sorted slice of keys from the given map.
func Keys[K cmp.Ordered, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
