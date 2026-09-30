//go:build goja_stats

package goja

import "sync/atomic"

type statsStruct struct {
	tinyClassTotal, tinyClassMisses, tinyObjectCreates, tinyObjectDeoptimizations atomic.Uint32

	tinyClassMultiTransitions atomic.Uint32
}

var stats statsStruct

func (s *statsStruct) incTinyClassTotal() {
	stats.tinyClassTotal.Add(1)
}

func (s *statsStruct) incTinyClassMisses() {
	stats.tinyClassMisses.Add(1)
}

func (s *statsStruct) incTinyObjectCreates() {
	stats.tinyObjectCreates.Add(1)
}

func (s *statsStruct) incTinyObjectDeoptimizations() {
	stats.tinyObjectDeoptimizations.Add(1)
}

func (s *statsStruct) incTinyClassMultiTransitions() {
	stats.tinyClassMultiTransitions.Add(1)
}

func printStats(printf func(string, ...any)) {
	tinyObjectMisses := stats.tinyClassMisses.Load()
	tinyObjectTotal := stats.tinyClassTotal.Load()
	tinyObjectCreates := stats.tinyObjectCreates.Load()
	tinyObjectDeoptimizations := stats.tinyObjectDeoptimizations.Load()
	tinyClassMultiTransitions := stats.tinyClassMultiTransitions.Load()

	printf("tinyObject miss ratio: %.02f%% (%d/%d)", float64(tinyObjectMisses)/float64(tinyObjectTotal)*100, tinyObjectMisses, tinyObjectTotal)
	printf("tinyObject deoptimizations ratio: %.02f%% (%d/%d)", float64(tinyObjectDeoptimizations)/float64(tinyObjectCreates)*100, tinyObjectDeoptimizations, tinyObjectCreates)
	printf("tinyClass multi property transitions: %d", tinyClassMultiTransitions)
}

func clearStats() {
	stats = statsStruct{}
}
