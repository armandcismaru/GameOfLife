package gol

import (
	"fmt"
	"net"
	"net/rpc"
	"os"
	"time"
)

var (
	//list of the workers that are connected to the engine
	workersList = make([]string, 1000)
	clients     = make([]*rpc.Client, 1000)
	n           = 0
	//the world with every paramether it has
	world         [][]byte
	width         int
	height        int
	turns         int
	requiredTurns int
	//lock chan is used as a lock to avoid race conditions
	lock = make(chan bool, 1)
	//channel to signal when to stop evolving the board and return the reult calculated so far
	stop     = make(chan bool)
	stopcont = make(chan bool)
	//variables required for being able to continue or start evolving a new board for the 'q' press
	engineListener net.Listener
	contrRes       *StatusReport
	contr          *rpc.Client
	visu           = false
	ct             = 0
	run            = false
)

type Engine struct{}

//function that will be called as a goroutine that splits the board between x and dx and sends a slice of that size to workers
//along with a left and a right slice that represent the neighbours of the newWorld slice
func startWorkers(ImageHeight, ImageWidth, workerID, startX, div, endX int, wrld [][]byte, calculateReport []WorkerReport, done chan bool) {
	left := make([]byte, ImageHeight)
	right := make([]byte, ImageHeight)
	newWorld := make([][]byte, ImageHeight)

	//create a strip with smaller size
	for k := range newWorld {
		newWorld[k] = make([]byte, endX-startX)
	}
	for k := range newWorld {
		for l := range newWorld[k] {
			newWorld[k][l] = wrld[k][l+workerID*div]
		}
	}
	l := startX - 1
	r := endX
	if l < 0 {
		l = ImageWidth - 1
	}
	if r > ImageWidth-1 {
		r = 0
	}
	//asign the neighbour slices
	for k := range left {
		left[k] = wrld[k][l]
		right[k] = wrld[k][r]
	}
	//create the neede request type and call a worker
	request := WorkerRequest{
		ImageHeight: ImageHeight,
		ImageWidth:  ImageWidth,
		Dx:          endX - startX,
		World:       newWorld,
		Left:        left,
		Right:       right,
	}

	err := clients[workerID].Call(CalculateNextState, request, &calculateReport[workerID])
	if err != nil {
		panic(err)
	}
	for calculateReport[workerID].Done != true {
	}
	//signal that the worker is done
	done <- calculateReport[workerID].Done
}

//function that will close the engine after 2 seconds
func closeEngine() {
	time.Sleep(2 * time.Second)
	os.Exit(0)

}

//close every worker connected to the engine then close the engine itself
func (b *Engine) CloseSystem(req bool, res *bool) (err error) {
	fmt.Println("Closing the system...")
	for i := 0; i < n; i++ {
		fmt.Println("Closing", i)
		var x bool
		err2 := clients[i].Call(CloseWorker, true, &x)
		if err2 != nil {
			fmt.Println(err2)
			panic(err2)
		}
	}
	go closeEngine()
	return nil
}

//register a worker by saving its IP and a poiter: *rpc.Client
func (b *Engine) Register(req RegisterWorker, res *StatusReport) (err error) {
	workersList[n] = req.WorkerAddres
	clients[n], _ = rpc.Dial("tcp", req.WorkerAddres)
	n++
	res.Turns = 0
	fmt.Println("Worker registered.", req.WorkerAddres)
	return nil
}

//The pause and unpause functions take advantage of the lock that is used to avoid race conditions, locking
func (b *Engine) Unpause(req bool, res *bool) (err error) {
	<-lock
	return nil
}

func (b *Engine) Pause(req bool, res *PauseReport) (err error) {
	res.Turns = turns
	lock <- true
	return nil
}

//function to disconnect the controller from the engine, it sets visualisation to false so the engine doesn't try to send information to a non existing controller
func (b *Engine) Disconnect(req bool, res *bool) (err error) {
	lock <- true
	visu = false
	<-lock
	return nil
}

//reconects the controller to the engine
func (b *Engine) ContinueSimulation(req ContinueRequest, res *StatusReport) (err error) {
	lock <- true
	ct++
	contrRes = res
	visu = req.Vis
	run = true
	if visu {
		var err3 error
		//recreates the visualisation connection if required
		contr, err3 = rpc.Dial("tcp", req.ControllerAddress)
		if err3 != nil {
			fmt.Println(err3)
			panic(err3)
		}
	}
	<-lock
	stp := false
	for turns < requiredTurns && !stp {
		select {
		case <-stopcont:
			stp = true
		default:
			time.Sleep(30 * time.Millisecond)
		}
	}

	contrRes.Turns = turns
	contrRes.World = world
	return nil
}

//stops the simulation by sending a stop signal through the channel
func (b *Engine) StopSimulation(req bool, res *StatusReport) (err error) {
	stop <- true
	if ct > 0 {
		ct = 0
		stopcont <- true
	}
	run = false
	res.World = world
	res.Turns = turns
	return nil
}

//closes the simulation if it is alreaedy running
func (n *Engine) CloseifRunning(req bool, res *bool) (err error) {
	if run {
		stop <- true
		if ct > 0 {
			stopcont <- true
		}
	}
	run = false
	return nil
}

//returns the state of the board
func (b *Engine) ReturnBoardState(req string, res *StatusReport) (err error) {
	lock <- true
	fmt.Println("ReturnBoardState")
	res.Turns = turns
	res.World = world
	<-lock
	return nil
}

//calculates and returns the number of alive cells
func (b *Engine) AliveCells(req string, res *AliveCellsReport) (err error) {
	lock <- true
	no := 0
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			if world[x][y] == alive {
				no++
			}
		}
	}
	res.Alive = no
	res.Turns = turns
	<-lock
	return nil
}

//starts the simulation
func (b *Engine) Start(req StartRequest, res *StatusReport) (err error) {
	//initialize all the required variables and starts work
	lock <- true
	run = true
	fmt.Println("Starting work...", n, req.ImageHeight, req.ImageWidth, req.Turns)
	visu = req.Visualisation
	requiredTurns = req.Turns
	contrRes = res
	if n == 0 {
		fmt.Println("No available workers.")
		return
	}
	world = make([][]byte, req.ImageHeight)
	for i := range world {
		world[i] = make([]byte, req.ImageWidth)
	}
	if req.Visualisation {
		var err3 error
		//creates a connection with the controller for visualisation (if visualisation is enabled)
		contr, err3 = rpc.Dial("tcp", req.ControllerAddress)
		if err3 != nil {
			fmt.Println(err3)
			panic(err3)
		}
	}
	<-lock
	copy(world, req.World)
	turns = 0
	height = req.ImageHeight
	width = req.ImageWidth
	done := make([]chan bool, n)
	for i := 0; i < n; i++ {
		done[i] = make(chan bool, 1)
	}
	stp := false

	//loop that evolves the turns
	for turns < req.Turns && !stp {
		select {
		//if it gets a stop signal, close the loop and return the results calculated so far
		case <-stop:
			stp = true
			break
		default:
			lock <- true
			div := height / n
			mod := height % n
			calculateReport := make([]WorkerReport, n)
			alCellsReport := new(AliveReport)
			i := 0
			//splits the board in pieces and calls the workers to process it
			for ; i < n-1; i++ {
				go startWorkers(req.ImageHeight, req.ImageWidth, i, i*div, div, (i+1)*div, world, calculateReport, done[i])
			}
			go startWorkers(req.ImageHeight, req.ImageWidth, i, i*div, div, (i+1)*div+mod, world, calculateReport, done[i])

			//checks if all workers are done
			for i := 0; i < n; i++ {
				<-done[i]
			}
			i = 0

			//reasembles the board
			for ; i < n-1; i++ {
				for y := 0; y < width; y++ {
					for x := i * div; x < (i+1)*div; x++ {
						world[y][x] = calculateReport[i].World[y][x-(i*div)]
					}
				}
			}
			for y := 0; y < width; y++ {
				for x := i * div; x < (i+1)*div+mod; x++ {
					world[y][x] = calculateReport[i].World[y][x-(i*div)]
				}
			}
			//sends the board data to the controller if visualisation is enabled
			if visu {
				clients[0].Call(CalculateAliveCells, VisualiseCellsRequest{req.ImageHeight, req.ImageWidth, 0, req.ImageWidth, world}, alCellsReport)
				var x bool
				contr.Call(Visualise, VisualiseRequest{alCellsReport.Cells, turns}, &x)
			}
			turns++
			<-lock
		}
	}

	lock <- true
	//returns the work done
	contrRes.Turns = turns
	contrRes.World = world
	<-lock
	run = false
	if ct > 0 {
		ct = 0
		stopcont <- true
	}
	fmt.Println("Work done. :)")
	return nil
}

//starts the listener and prints the IPs
func Eng(port string) {
	rpc.Register(&Engine{})
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

	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		fmt.Println(err)
	}
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	conn.Close()

	fmt.Println("Engine local address: ", add+":"+port)
	fmt.Println("Engine internet address: ", localAddr.IP.To16().String()+":"+port)
	engineListener, _ = net.Listen("tcp", ":"+port)
	rpc.Accept(engineListener)
	defer engineListener.Close()
}
