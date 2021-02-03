package gol

import (
	"fmt"
	"strconv"
	"time"

	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events    chan<- Event
	ioCommand chan<- ioCommand
	ioIdle    <-chan bool
}

const alive = 255
const dead = 0

func mod(x, m int) int {
	return (x + m) % m
}

// Returns the number of alive neighbours of a given pixel.
func calculateNeighbours(p Params, x, y int, world [][]byte) int {
	neighbours := 0
	for i := -1; i <= 1; i++ {
		for j := -1; j <= 1; j++ {
			if i != 0 || j != 0 {
				if world[mod(y+i, p.ImageHeight)][mod(x+j, p.ImageWidth)] == alive {
					neighbours++
				}
			}
		}
	}
	return neighbours
}

// Calculates the next state of a given board and outputs to a newly created 2D slice.
func calculateNextState(startX, endX int, p Params, world [][]byte) [][]byte {
	newWorld := make([][]byte, p.ImageHeight)
	for i := range newWorld {
		newWorld[i] = make([]byte, p.ImageWidth)
	}
	for y := 0; y < p.ImageHeight; y++ {
		for x := startX; x < endX; x++ {
			neighbours := calculateNeighbours(p, x, y, world)
			if world[y][x] == alive {
				if neighbours == 2 || neighbours == 3 {
					newWorld[y][x] = alive
				} else {
					newWorld[y][x] = dead
				}
			} else {
				if neighbours == 3 {
					newWorld[y][x] = alive
				} else {
					newWorld[y][x] = dead
				}
			}
		}
	}
	return newWorld
}

// Calculates the number of alive cells from a given board.
func calculateAliveCells(p Params, world [][]byte) []util.Cell {
	aliveCells := []util.Cell{}

	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			if world[y][x] == alive {
				aliveCells = append(aliveCells, util.Cell{X: x, Y: y})
			}
		}
	}
	return aliveCells
}

// Creates a PGM image to output.
func makePGM(p Params, filename chan string, c distributorChannels, turn int, ioOut chan<- uint8, world [][]byte) {
	filename <- strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(turn)

	c.ioCommand <- ioOutput
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			ioOut <- world[y][x]
		}
	}
}

// modified from https://stackoverflow.com/questions/19374219/how-to-find-the-difference-between-two-slices-of-strings
func difference(a, b []util.Cell) []util.Cell {
	temp := map[util.Cell]int{}
	for _, s := range a {
		temp[s]++
	}
	for _, s := range b {
		temp[s]--
	}

	var result []util.Cell
	for s, v := range temp {
		if v != 0 {
			result = append(result, s)
		}
	}
	return result
}

// Worker method which advances the board and notifies the distributor when its finished.
func worker(startX, endX int, turnDone chan<- bool, outWorld chan<- [][]byte, world [][]byte, p Params) {
	newWorld := calculateNextState(startX, endX, p, world)
	outWorld <- newWorld
	turnDone <- true
}

// Distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels, ioIn <-chan uint8, ioOut chan<- uint8, filename chan string, keyPresses <-chan rune) {
	c.ioCommand <- ioInput

	// Create a 2D slice to store the world.
	world := make([][]byte, p.ImageHeight)

	for i := range world {
		world[i] = make([]byte, p.ImageWidth)
	}

	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			world[y][x] = <-ioIn
		}
	}

	// For all initially alive cells send a CellFlipped Event.
	turn := 0
	aliveCells := calculateAliveCells(p, world)
	for i := range aliveCells {
		var CellFlipped = CellFlipped{
			CompletedTurns: turn,
			Cell: util.Cell{
				X: aliveCells[i].X,
				Y: aliveCells[i].Y,
			},
		}
		c.events <- CellFlipped
	}

	// Execute all turns of the Game of Life.
	// Send correct Events when required, e.g. CellFlipped, TurnComplete and FinalTurnComplete.
	done := make(chan bool, p.Threads)
	outWorld := make([]chan [][]byte, p.Threads)

	for i := 0; i < p.Threads; i++ {
		outWorld[i] = make(chan [][]byte, 1)
	}

	ticker := time.NewTicker(2 * time.Second)
	div := p.ImageWidth / p.Threads
	mod := p.ImageWidth % p.Threads

	for turn < p.Turns {
		select {
		case <-ticker.C:
			cells := len(calculateAliveCells(p, world))
			c.events <- AliveCellsCount{turn, cells}
		default:
			select {
			case key := <-keyPresses:
				switch key {
				case 's':
					makePGM(p, filename, c, turn, ioOut, world)
				case 'q':
					makePGM(p, filename, c, turn, ioOut, world)
					c.ioCommand <- ioCheckIdle
					<-c.ioIdle
					c.events <- StateChange{turn, Quitting}
					time.Sleep(500 * time.Millisecond)
					close(c.events)
				case 'p':
					fmt.Println("Game is being paused on turn:", turn)
					for i := 0; i == 0; {
						select {
						case press := <-keyPresses:
							if press == 'p' {
								i = 1
								fmt.Println("Continuing")
								break
							}
						default:
							time.Sleep(100 * time.Millisecond)
						}
					}
				}
			default:
				i := 0
				for i = 0; i < p.Threads-1; i++ {
					go worker(i*div, (i+1)*div, done, outWorld[i], world, p)
				}
				go worker(i*div, (i+1)*div+mod, done, outWorld[i], world, p)

				for i := 0; i < p.Threads; i++ {
					<-done
				}
				for i := 0; i < p.Threads-1; i++ {
					newThreadSlice := <-outWorld[i]
					for y := 0; y < p.ImageWidth; y++ {
						for x := i * div; x < (i+1)*div; x++ {
							world[y][x] = newThreadSlice[y][x]
						}
					}
				}
				newThreadSlice := <-outWorld[i]
				for y := 0; y < p.ImageWidth; y++ {
					for x := i * div; x < (i+1)*div+mod; x++ {
						world[y][x] = newThreadSlice[y][x]
					}
				}
				x := calculateAliveCells(p, world)
				cells := difference(aliveCells, x)

				for i := range cells {
					var CellFlipped = CellFlipped{
						CompletedTurns: turn,
						Cell: util.Cell{
							X: cells[i].X,
							Y: cells[i].Y,
						},
					}
					c.events <- CellFlipped
				}

				turn++
				c.events <- TurnComplete{
					CompletedTurns: turn,
				}
				aliveCells = x
			}
		}
	}
	c.events <- FinalTurnComplete{
		CompletedTurns: turn,
		Alive:          aliveCells,
	}

	// Logic to output the state of the board as a PGM image
	makePGM(p, filename, c, turn, ioOut, world)

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
