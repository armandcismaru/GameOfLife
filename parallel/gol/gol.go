package gol

import "strconv"

// Params provides the details of how to run the Game of Life and which image to load.
type Params struct {
	Turns       int
	Threads     int
	ImageWidth  int
	ImageHeight int
}

// Run starts the processing of Game of Life. It should initialise channels and goroutines.
func Run(p Params, events chan<- Event, keyPresses <-chan rune) {

	ioCommand := make(chan ioCommand)
	ioIdle := make(chan bool)

	distributorChannels := distributorChannels{
		events,
		ioCommand,
		ioIdle,
	}
	output := make(chan uint8)
	input := make(chan uint8)
	filename := make(chan string, 1)

	filename <- strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.ImageWidth)

	ioChannels := ioChannels{
		command:  ioCommand,
		idle:     ioIdle,
		filename: filename,
		output:   output,
		input:    input,
	}
	go startIo(p, ioChannels)
	go distributor(p, distributorChannels, input, output, filename, keyPresses)
}
