package gol

import (
	"fmt"
	"net"
	"net/rpc"
	"os"
	"strconv"
	"time"

	"uk.ac.bris.cs/gameoflife/util"
)

var (
	visualiseChannel controllerChannels
	aliv             []util.Cell
)

type Controller struct{}
type controllerChannels struct {
	events    chan<- Event
	ioCommand chan<- ioCommand
	ioIdle    <-chan bool
}

//send the filename and the output command plus the world in order to output a pgm file
func genPgm(filename chan string, p Params, name string, c controllerChannels, ioOut chan<- uint8, world [][]byte) {
	filename <- strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.ImageWidth) + "x" + name
	c.ioCommand <- ioOutput
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			ioOut <- world[y][x]
		}
	}
}

//calculate the alive cells
func calculateAliveCells(p Params, world [][]byte) []util.Cell {
	aliveCells := []util.Cell{}

	for y := range world {
		for x := range world[y] {
			if world[y][x] == alive {
				aliveCells = append(aliveCells, util.Cell{X: x, Y: y})
			}
		}
	}

	return aliveCells
}

// function that is called as a goroutine, sends a request for the alive cells every two seconds
func reqAliveCount(c controllerChannels, client *rpc.Client, close chan bool, ticker *time.Ticker) {
	req1 := "req"
	var cells int
	var turn int
	res2 := AliveCellsReport{cells, turn}
	for {
		select {
		//if we get a signal through the close channel, we will stop the function
		case <-close:
			return
		default:
			select {
			case <-ticker.C:
				err := client.Call(AliveCells, req1, &res2)
				if err != nil {
					panic(err)
				}
				c.events <- AliveCellsCount{res2.Turns, res2.Alive}
			default:
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
}

// function that is called as a goroutine, checks for key presses
func keyCheck(p Params, client *rpc.Client, keyPresses <-chan rune, filename chan string, ioOut chan<- uint8, c controllerChannels, close chan bool, ticker *time.Ticker, save chan StatusReport) {
	for {
		select {
		case <-close:
			return
		case key := <-keyPresses:
			switch key {
			case 's':
				res := new(StatusReport)
				client.Call(ReturnBoardState, "req", &res)
				genPgm(filename, p, strconv.Itoa(res.Turns), c, ioOut, res.World)
			case 'q':
				res := true
				client.Call(Disconnect, true, &res)
				os.Exit(0)

			case 'p':
				//stop the timer so it doesn't thick when the calculation is paused
				ticker.Stop()
				res := new(PauseReport)
				client.Call(Pause, true, &res)
				fmt.Println("Paussed on turn : ", res.Turns)
				for i := 0; i == 0; {
					select {
					case k := <-keyPresses:
						if k == 'p' {
							fmt.Println("Continuing")
							//reset the timer so it starts ticking again
							ticker.Reset(2 * time.Second)
							var x bool
							client.Call(Unpause, true, &x)
							i = 1
							break
						}
					default:
						time.Sleep(100 * time.Millisecond)
					}
				}
			case 'k':
				res := true
				ticker.Stop()
				//first we stop the simulation in order to return the calculated value so far and then we close the system
				r := new(StatusReport)
				client.Call(StopSimulation, true, &r)
				save <- *r
				client.Call(CloseSystem, true, &res)

			}
		}
	}
}

//modified from https://stackoverflow.com/questions/19374219/how-to-find-the-difference-between-two-slices-of-strings
//used to calculate the difference between two slices
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

//sends the required events to sdl in order to be able to visualise the board state (if visualise is enabled)
func (*Controller) Visualise(req VisualiseRequest, res *bool) (err error) {
	q := difference(aliv, req.Cells)
	for i := range q {
		var CellFlipped = CellFlipped{
			CompletedTurns: req.Turns,
			Cell: util.Cell{
				X: q[i].X,
				Y: q[i].Y,
			},
		}
		visualiseChannel.events <- CellFlipped
	}
	visualiseChannel.events <- TurnComplete{req.Turns}
	aliv = req.Cells
	return
}

//listener required for accepting rpc calls for visualise (only when visualise is enabled)
func createListenerContr(port string) {
	rpc.Register(&Controller{})
	engineListener, _ = net.Listen("tcp", ":"+port)
	rpc.Accept(engineListener)

}

// Controller works as a controller, communicating with the engine, sending work and receiving the results
func controller(p Params, c controllerChannels, ioIn <-chan uint8, ioOut chan<- uint8, keyPresses <-chan rune, filename chan string, engineAddress, controllerPort string, vis, cont bool) {
	//create the listener if visualise is enabled
	if vis {
		visualiseChannel = c
		go createListenerContr(controllerPort)
	}
	res := new(StatusReport)
	client, _ := rpc.Dial("tcp", engineAddress)
	ticker := time.NewTicker(2 * time.Second)
	quitCellRequest := make(chan bool)
	quitKeyCheck := make(chan bool)

	//get the internet and local ip address of the component
	name, errc := os.Hostname()
	if errc != nil {
		fmt.Println(errc)
		return
	}
	addrs, errc := net.LookupHost(name)
	if errc != nil {
		fmt.Println(errc)
		return
	}
	var add string
	for _, a := range addrs {
		add = a
	}
	save := make(chan StatusReport, 1)

	go keyCheck(p, client, keyPresses, filename, ioOut, c, quitKeyCheck, ticker, save)
	go reqAliveCount(c, client, quitCellRequest, ticker)

	//if we want to continue the previous work we need to call a different function through rpc
	if cont {
		time.Sleep(30 * time.Millisecond)
		client.Call(ContinueSimulation, ContinueRequest{p.Turns, add + ":" + controllerPort, vis}, res)
	} else {
		var r bool

		//close the previous simulation if there is one running
		errc := client.Call(CloseifRunning, true, &r)
		if errc != nil {
			fmt.Println(errc)
			panic(errc)
		}

		//read the world from the input pgm file
		filename <- strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.ImageWidth)
		c.ioCommand <- ioInput
		world := make([][]byte, p.ImageHeight)
		for i := range world {
			world[i] = make([]byte, p.ImageWidth)
		}
		for y := 0; y < p.ImageHeight; y++ {
			for x := 0; x < p.ImageWidth; x++ {
				world[y][x] = <-ioIn
			}
		}

		//creating the required request and calling the engine to start evolving the board
		req := StartRequest{
			Turns:             p.Turns,
			ImageHeight:       p.ImageHeight,
			ImageWidth:        p.ImageWidth,
			World:             world,
			ControllerAddress: add + ":" + controllerPort,
			Visualisation:     vis,
		}
		fmt.Println(add + ":" + controllerPort)
		client.Call(Start, req, res)
	}
	quitCellRequest <- true
	quitKeyCheck <- true
	client.Close()
	//required in some instances to wait for everything to finish calculations and close
	turn := 0
	if len(res.World) == 0 {
		x := <-save
		turn = x.Turns
		world = x.World
	} else {
		world = res.World
		turn = res.Turns
	}
	c.events <- FinalTurnComplete{
		CompletedTurns: turn,
		Alive:          calculateAliveCells(p, world),
	}

	genPgm(filename, p, strconv.Itoa(turn), c, ioOut, world)
	// Make sure that the Io has finished any output before exiting.

	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}
	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
