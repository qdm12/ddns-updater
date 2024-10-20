package info

import (
	"hash/maphash"
	"math/rand"
)

var _ rand.Source = new(mapHashSource)

type mapHashSource struct{}

func (s *mapHashSource) Int63() int64 {
	v := new(maphash.Hash).Sum64()
	return int64(v >> 1) //nolint:gosec
}

func (s *mapHashSource) Seed(_ int64) {}
