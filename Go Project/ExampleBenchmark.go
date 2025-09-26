package main

import (
	"fmt"
	"time"
)

var accuracyList = make([]float32, 0, 250)
var durationMap = make(map[int][]time.Duration)

func benchmarkComputationalCost() {
	explainabilityVerbose = false

	fmt.Printf("Executing with observers...\n")
	spawnObservers = 1
	benchmarkRunComputationalCost()

	fmt.Printf("Executing without observers...\n")
	spawnObservers = 0
	benchmarkRunComputationalCost()
}

func benchmarkRunComputationalCost() {
	durationMap = make(map[int][]time.Duration)
	for range 5 {
		benchmarkSetup(50, 1000, 5)
		benchmarkSetup(100, 1000, 5)
		benchmarkSetup(150, 1000, 5)
		benchmarkSetup(200, 1000, 5)
		benchmarkSetup(250, 1000, 5)
		benchmarkSetup(300, 1000, 5)
		benchmarkSetup(350, 1000, 5)
		benchmarkSetup(400, 1000, 5)
		benchmarkSetup(450, 1000, 5)
		benchmarkSetup(500, 1000, 5)
	}
	fmt.Printf("Average Durations: [")
	for key := range durationMap {
		totalTime := durationMap[key][0]
		for i, t := range durationMap[key] {
			if i == 0 {
				continue
			}
			totalTime += t
		}
		avrgTime := totalTime / 5
		fmt.Printf("%v\t", avrgTime)
	}
	fmt.Printf("]\n")
}

func benchmarkFidelity() {
	spawnObservers = 1
	benchmarkRunFidelity()

	fmt.Printf("Accuracies: [")
	for _, accuracy := range accuracyList {
		fmt.Printf("%v\t", accuracy)
	}
	fmt.Printf("]\n")
}

func benchmarkRunFidelity() {
	benchmarkSetup(100, 1000, 5)
	benchmarkSetup(100, 1000, 10)
	benchmarkSetup(250, 1000, 5)
	benchmarkSetup(250, 1000, 10)

	benchmarkSetup(100, 250, 5)
	benchmarkSetup(100, 250, 10)
	benchmarkSetup(250, 250, 5)
	benchmarkSetup(250, 250, 10)
}

func benchmarkSetup(numAgents, bandwidth, messageGroups int) {
	start := time.Now()
	fmt.Printf("Simulating for (%v) Model agents, (%v) bandwidth, (%v) message groups\n", numAgents, bandwidth, messageGroups)
	numClusters = messageGroups
	serv := CreateHelloMetaServer(numAgents, 10, 100*time.Millisecond, bandwidth)
	serv.Start()
	end := time.Now()
	//fmt.Printf("Duration : (%v)\n", end.Sub(start))
	_, ok := durationMap[numAgents]
	if !ok {
		durationMap[numAgents] = []time.Duration{end.Sub(start)}
	} else {
		durationMap[numAgents] = append(durationMap[numAgents], end.Sub(start))
	}
	serv = nil
	/*
		fmt.Printf("Accuracies: [")
		for _, accuracy := range accuracyList {
			fmt.Printf("%v\t", accuracy)
		}
		fmt.Printf("]\n")
		accuracyList = accuracyList[:0]
	*/
}
