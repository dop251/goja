//go:build !goja_stats

package goja

type statsStruct struct{}

var stats statsStruct

func (s *statsStruct) incTinyClassTotal() {
}

func (s *statsStruct) incTinyClassMisses() {
}

func (s *statsStruct) incTinyObjectCreates() {
}

func (s *statsStruct) incTinyObjectDeoptimizations() {
}

func (s *statsStruct) incTinyClassMultiTransitions() {
}

func printStats(printf func(string, ...any)) {
}

func clearStats() {
}
