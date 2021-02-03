package main

import (
	"fmt"
	"os"
	"testing"

	"uk.ac.bris.cs/gameoflife/gol"
)

// Bench test with hardcoded parameters.
func BenchmarkGol(b *testing.B) {
	p := gol.Params{
		Turns:       100,
		ImageWidth:  64,
		ImageHeight: 64,
	}
	for i := 1; i <= 16; i++ {
		p.Threads = i
		testName := fmt.Sprintf("%dx%dx%d-%d", p.ImageWidth, p.ImageHeight, p.Turns, p.Threads)
		b.Run(testName, func(b *testing.B) {
			os.Stdout = nil
			events := make(chan gol.Event)
			gol.Run(p, events, nil)
			for event := range events {
				switch event.(type) {
				case gol.FinalTurnComplete:
					break
				}
			}
		})
	}
}
