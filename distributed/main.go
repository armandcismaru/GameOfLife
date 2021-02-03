package main

import (
	"flag"
	"fmt"
	"runtime"

	"uk.ac.bris.cs/gameoflife/gol"
	"uk.ac.bris.cs/gameoflife/sdl"
)

// main is the function called when starting Game of Life with 'go run .'
func main() {
	runtime.LockOSThread()
	var params gol.Params
	var port string
	var typ string
	var engineAddress string
	var visualise bool
	var con bool

	flag.IntVar(
		&params.Threads,
		"t",
		1,
		"Specify the number of worker threads to use for each worker. Defaults to 1.")
	flag.IntVar(
		&params.ImageWidth,
		"w",
		512,
		"Specify the width of the image. Defaults to 512.")
	flag.IntVar(
		&params.ImageHeight,
		"h",
		512,
		"Specify the height of the image. Defaults to 512.")
	flag.IntVar(
		&params.Turns,
		"turns",
		10000000000,
		"Specify the number of turns to process. Defaults to 10000000000.")
	flag.StringVar(&port,
		"Port",
		"8030",
		"Specify the port the component will use.")
	flag.StringVar(&typ,
		"Type",
		"controller",
		"Specify what type of instance you are starting.")
	flag.StringVar(&engineAddress,
		"EngineAddress",
		"127.0.0.1:8040",
		"Specify the address of the engine.")
	flag.BoolVar(&visualise,
		"Visualise",
		false,
		"Specify if the controller should dislpay the progress of the board (high impact on speed). Defaults to false.",
	)
	flag.BoolVar(&con,
		"Continue",
		false,
		"Specify if the controller should Continue the progress of the board. Defaults to false.",
	)
	flag.Parse()

	keyPresses := make(chan rune, 10)
	events := make(chan gol.Event, 1000)

	if typ == "controller" {
		fmt.Println("Controller")
		// setVars will pass the flags given by the user (workaround to not modify the Run() function)
		gol.SetVars(engineAddress, port, visualise, con)
		gol.Run(params, events, keyPresses)
		sdl.Start(params, events, keyPresses)
	} else if typ == "Engine" {
		//start the engine
		fmt.Println("Engine")
		gol.Eng(port)

	} else if typ == "Worker" {
		//start the worker
		fmt.Println("Worker")
		gol.Work(port, engineAddress, params.Threads)
	} else {
		fmt.Println("Invalid inputs...")
	}
}
