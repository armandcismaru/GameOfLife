package gol

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"time"

	"uk.ac.bris.cs/gameoflife/util"
)

const alive = 255
const dead = 0

var (
	workerListener net.Listener
	threads        int = 1
)

type Worker struct{}

//closes the worker after 4 seconds
func closeWorker() {
	time.Sleep(4 * time.Second)
	os.Exit(0)
}

//function that is called when closing the workers
func (*Worker) CloseWorker(req bool, res *bool) (err error) {
	fmt.Println("Closing worker...")
	go closeWorker()
	return nil
}

//returns the alive cells in a slice of util.Cells
func (*Worker) CalculateAliveCells(req VisualiseCellsRequest, res *AliveReport) (err error) {
	for y := range req.World {
		for x := range req.World[y] {
			if req.World[y][x] == alive {
				res.Cells = append(res.Cells, util.Cell{X: x, Y: y})
			}
		}
	}
	return nil
}

//function called as a goroutine, calculates the next state between y and dy
func calculateNextState(world [][]byte, dx, y, dy, ImageHeight, ImWidth int, left, right []byte, outWorld chan [][]byte, done chan bool) {
	newWorld := make([][]byte, ImageHeight)
	for i := range newWorld {
		newWorld[i] = make([]byte, dx)
	}
	for j := 1; j < dx-1; j++ {
		for i := 0; i < ImageHeight; i++ {

			a := i - 1
			b := j - 1
			x := i + 1
			y := j + 1

			if a == -1 {
				a = ImageHeight - 1
			}

			if x == ImageHeight {
				x = 0
			}
			aliveNeighbours := int(world[a][b]) + int(world[a][j]) + int(world[a][y]) + int(world[i][y]) + int(world[x][y]) + int(world[x][j]) + int(world[x][b]) + int(world[i][b])
			aliveNeighbours /= alive
			if world[i][j] == alive {
				if aliveNeighbours < 2 || aliveNeighbours > 3 {
					newWorld[i][j] = dead
				} else {
					newWorld[i][j] = alive
				}
			} else {
				if world[i][j] == dead {
					if aliveNeighbours == 3 {
						newWorld[i][j] = alive
					}
				}
			}

		}
	}

	for i := 0; i < ImageHeight; i++ {
		a := i - 1
		x := i + 1
		if a == -1 {
			a = ImageHeight - 1
		}

		if x == ImageHeight {
			x = 0
		}
		//when at edges, we need to look at the left and right slice
		aliveNeighbours := int(left[a]) + int(world[a][0]) + int(world[a][1]) + int(world[i][1]) + int(world[x][1]) + int(world[x][0]) + int(left[x]) + int(left[i])
		aliveNeighbours /= alive
		if world[i][0] == alive {
			if aliveNeighbours < 2 || aliveNeighbours > 3 {
				newWorld[i][0] = dead
			} else {
				newWorld[i][0] = alive
			}
		} else {
			if world[i][0] == dead {
				if aliveNeighbours == 3 {
					newWorld[i][0] = alive
				}
			}

		}

		aliveNeighbours = int(right[a]) + int(world[a][dx-2]) + int(world[a][dx-1]) + int(world[i][dx-2]) + int(world[x][dx-1]) + int(world[x][dx-2]) + int(right[x]) + int(right[i])
		aliveNeighbours /= alive
		if world[i][dx-1] == alive {
			if aliveNeighbours < 2 || aliveNeighbours > 3 {
				newWorld[i][dx-1] = dead
			} else {
				newWorld[i][dx-1] = alive
			}
		} else {
			if world[i][dx-1] == dead {
				if aliveNeighbours == 3 {
					newWorld[i][dx-1] = alive
				}
			}

		}
	}
	outWorld <- newWorld
	done <- true
}

//function that is called through rpc
//takes the piece of the board it recieves plus its neighbours and splits it depending of the number of threads it has available, returning the next state of the slice
func (*Worker) CalculateNextState(req WorkerRequest, res *WorkerReport) (err error) {
	div := req.ImageHeight / threads
	mod := req.ImageHeight % threads
	i := 0
	outWorld := make([]chan [][]byte, threads)
	done := make([]chan bool, threads)
	nworld := make([][]byte, req.ImageHeight)
	for j := range nworld {
		nworld[j] = make([]byte, req.ImageWidth)
	}

	for i = 0; i < threads; i++ {
		outWorld[i] = make(chan [][]byte, 1)
		done[i] = make(chan bool)
	}
	//start all the goroutines depending on the number of threads
	for i = 0; i < threads-1; i++ {
		go calculateNextState(req.World, req.Dx, i*div, (i+1)*div, req.ImageHeight, req.ImageWidth, req.Left, req.Right, outWorld[i], done[i])
	}
	go calculateNextState(req.World, req.Dx, i*div, (i+1)*div+mod, req.ImageHeight, req.ImageWidth, req.Left, req.Right, outWorld[i], done[i])
	//waits for every thread to finish
	for i = 0; i < threads; i++ {
		<-done[i]
	}
	//reassembles the board
	for i = 0; i < threads-1; i++ {
		newWorld := <-outWorld[i]
		for y := i * div; y < (i+1)*div; y++ {
			for x := 0; x < req.Dx; x++ {
				nworld[y][x] = newWorld[y][x]
			}
		}
	}
	newWorld := <-outWorld[i]
	for y := i * div; y < (i+1)*div+mod; y++ {
		for x := 0; x < req.Dx; x++ {
			nworld[y][x] = newWorld[y][x]
		}
	}
	res.World = nworld
	res.Done = true
	return nil
}

//creates the listener of the worker
func createListener(port string, done chan bool) {
	rpc.Register(&Worker{})
	workerListener, _ = net.Listen("tcp", ":"+port)
	rpc.Accept(workerListener)
	fmt.Println("yes")
	done <- true
}

//starts the listener and registers on the engine
func Work(port string, engineAddr string, thr int) {
	threads = thr
	done := make(chan bool)
	go createListener(port, done)

	client, _ := rpc.Dial("tcp", engineAddr)

	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	conn.Close()

	name, err := os.Hostname()
	if err != nil {
		fmt.Println(err)
		return
	}
	addrs, err := net.LookupHost(name)
	if err != nil {
		fmt.Println(err)
		return
	}
	var add string
	for _, a := range addrs {
		add = a
	}

	fmt.Println("Worker local address:" + add + ":" + port)
	fmt.Println("Worker internet address:" + localAddr.IP.To4().String() + ":" + port)
	fmt.Println("Threads:", thr)

	regs := RegisterWorker{add + ":" + port}
	status := new(StatusReport)
	client.Call(Register, regs, status)
	fmt.Println("Worker registered.")

	<-done
}
