package main

import (
	"fmt"
	"os"
	"testing"

	"uk.ac.bris.cs/gameoflife/gol"
)

func BenchmarkGol(b *testing.B) {
	var pr gol.Params
	tests := []gol.Params{
		{ImageWidth: 512, ImageHeight: 512},
		{ImageWidth: 5120, ImageHeight: 5120},
	}
	for i := range tests {
		turns := []int{100, 200}
		for j := range turns {
			pr.ImageWidth = tests[i].ImageWidth
			pr.ImageHeight = tests[i].ImageHeight
			pr.Turns = turns[j]
			testName := fmt.Sprintf("%dx%dx%d-%d", pr.ImageWidth, pr.ImageHeight, pr.Turns, 4)
			b.Run(testName, func(b *testing.B) {
				os.Stdout = nil
				events := make(chan gol.Event)
				gol.Run(pr, events, nil)
				for event := range events {
					switch event.(type) {
					case gol.FinalTurnComplete:
						break

					}
				}
			})
		}
	}
}
