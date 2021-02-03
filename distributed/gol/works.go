package gol

import "uk.ac.bris.cs/gameoflife/util"

var Register = "Engine.Register"
var Start = "Engine.Start"
var AliveCells = "Engine.AliveCells"
var ReturnBoardState = "Engine.ReturnBoardState"
var StopSimulation = "Engine.StopSimulation"
var Pause = "Engine.Pause"
var Unpause = "Engine.Unpause"
var CloseSystem = "Engine.CloseSystem"
var Disconnect = "Engine.Disconnect"
var ContinueSimulation = "Engine.ContinueSimulation"
var CalculateNextState = "Worker.CalculateNextState"
var CloseWorker = "Worker.CloseWorker"
var CalculateAliveCells = "Worker.CalculateAliveCells"
var Visualise = "Controller.Visualise"
var CloseifRunning = "Engine.CloseifRunning"

type VisualiseCellsRequest struct {
	ImageHeight int
	ImWidth     int
	X           int
	Dx          int
	World       [][]byte
}

type StartRequest struct {
	ImageHeight       int
	ImageWidth        int
	Turns             int
	World             [][]byte
	ControllerAddress string
	Visualisation     bool
}

type WorkerReport struct {
	World [][]byte
	Done  bool
}

type RegisterWorker struct {
	WorkerAddres string
}

type StatusReport struct {
	Turns int
	World [][]byte
}

type AliveReport struct {
	Cells []util.Cell
}

type AliveCellsReport struct {
	Alive int
	Turns int
}

type PauseReport struct {
	Turns int
}

type WorkerRequest struct {
	ImageHeight int
	ImageWidth  int
	Dx          int
	World       [][]byte
	Left        []byte
	Right       []byte
}

type VisualiseRequest struct {
	Cells []util.Cell
	Turns int
}

type ContinueRequest struct {
	Turns             int
	ControllerAddress string
	Vis               bool
}
