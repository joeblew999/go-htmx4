//go:build !js

package ratelimit

// Allow always allows outside Workers (standard Go has no bindings) and reports ErrNoBinding.
func Allow(binding, key string) (bool, error) { return true, ErrNoBinding }
