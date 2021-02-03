package gol

var (
	engineAddr     string = "127.0.0.1:8040"
	controllerPort string = "8030"
	visualise      bool   = false
	contin         bool   = false
)

// Params provides the details of how to run the Game of Life and which image to load.
type Params struct {
	Turns       int
	Threads     int
	ImageWidth  int
	ImageHeight int
}

//modify the values that are given through flags
func SetVars(engAddr string, cPort string, vis bool, cont bool) {
	engineAddr = engAddr
	controllerPort = cPort
	visualise = vis
	contin = cont
}

// Run starts the processing of Game of Life. It should initialise channels and goroutines.
func Run(pa Params, events chan<- Event, keyPresses <-chan rune) {

	ioCommand := make(chan ioCommand)
	ioIdle := make(chan bool)

	controllerChannels := controllerChannels{
		events,
		ioCommand,
		ioIdle,
	}
	//create channels for io
	out := make(chan uint8)
	in := make(chan uint8)
	filename := make(chan string, 2)
	ioChannels := ioChannels{
		command:  ioCommand,
		idle:     ioIdle,
		filename: filename,
		output:   out,
		input:    in,
	}

	go controller(pa, controllerChannels, in, out, keyPresses, filename, engineAddr, controllerPort, visualise, contin)
	go startIo(pa, ioChannels)
}
