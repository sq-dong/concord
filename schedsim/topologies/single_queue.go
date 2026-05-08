package topologies

import (
	"fmt"

	"github.com/epfl-dcsl/schedsim/blocks"
	"github.com/epfl-dcsl/schedsim/engine"
)

// SingleQueue implement a single-generator-multiprocessor topology with a single
// queue. Each processor just dequeues from this queue
func SingleQueue(lambda, mu, duration, quantum, stddev, ctxCost float64, genType, procType int) {

	engine.InitSim()

	// Add generator, statistics
	var g blocks.Generator
	var stats *blocks.DetailedKeeper

	if genType == 0 {
		g = blocks.NewMMRandGenerator(lambda, mu)
		stats = blocks.NewDetailedKeeper(1, []float64{1/mu})
	} else if genType == 1 {
		g = blocks.NewMDRandGenerator(lambda, 1/mu)
		stats = blocks.NewDetailedKeeper(1, []float64{1/mu})
	} else if genType == 2 {
		g = blocks.NewMBRandGenerator(lambda, 1, 10*(1/mu-0.9), 0.9)
		stats = blocks.NewDetailedKeeper(2, []float64{1,10*(1/mu-0.9)})
	} else if genType == 3 {
		g = blocks.NewMBRandGenerator(lambda, 0.5, 200*(1/mu-0.995*0.5), 0.995)
		stats = blocks.NewDetailedKeeper(2, []float64{0.5,200*(1/mu-0.995*0.5)})
	} else if genType == 4 {
		g = blocks.NewMBRandGenerator(lambda, 1, 1000*(1/mu-0.999), 0.999)
		stats = blocks.NewDetailedKeeper(2, []float64{1,1000*(1/mu-0.999)})
	} else if genType == 5 {
		g = blocks.NewMBRandGenerator(lambda, 1, 2*(1/mu-0.5), 0.5)
		stats = blocks.NewDetailedKeeper(2, []float64{1,2*(1/mu-0.5)})
	}

	//Init the statistics
	stats.SetName("Main Stats")
	engine.InitStats(stats)

	g.SetCreator(&blocks.SimpleReqCreator{})

	// Create queues
	q := blocks.NewQueue()

	// Create processors

	if procType == 0 {
		for i := 0; i < cores; i++ {
			p := &blocks.RTCProcessor{}
			p.AddInQueue(q)
			p.SetReqDrain(stats)
			engine.RegisterActor(p)
		}
	} else if procType == 1 {
		p := blocks.NewPSProcessor()
		p.SetWorkerCount(cores)
		p.AddInQueue(q)
		p.SetReqDrain(stats)
		engine.RegisterActor(p)
	} else if procType == 2 {
		for i := 0; i < cores; i++ {
			p := blocks.NewTSProcessor(quantum,stddev)
			p.SetCtxCost(ctxCost)
			p.AddInQueue(q)
			p.SetReqDrain(stats)
			engine.RegisterActor(p)
		}
	}

	g.AddOutQueue(q)

	// Register the generator
	engine.RegisterActor(g)

	fmt.Printf("Cores:%v\tservice_rate:%v\tinterarrival_rate:%v\n", cores, mu, lambda)
	engine.Run(duration)
}
