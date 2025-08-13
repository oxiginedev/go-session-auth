package utils

import (
	"reflect"
	"strings"
)

// SliceContains checks if an item exists in slice
func SliceContains[T comparable](haystack []T, needle T) bool {
	for _, hh := range haystack {
		ht := reflect.TypeOf(hh)
		if ht.Kind() == reflect.String {
			if strings.EqualFold(reflect.ValueOf(hh).String(),
				reflect.ValueOf(needle).String()) {
				return true
			}
			continue
		}

		if needle == hh {
			return true
		}
	}

	return false
}
