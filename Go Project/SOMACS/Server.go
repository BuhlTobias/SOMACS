package SOMACS

import (
	"fmt"
	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/server"
	"github.com/google/uuid"
	"slices"
	"time"
)

type Server struct {
	*server.BaseServer[IGenericAgent]
	modelAgents    []uuid.UUID
	observerAgents []uuid.UUID
	metaAgents     []uuid.UUID

	modelAgentMap    map[uuid.UUID]*ModelAgent
	metaAgentMap     map[uuid.UUID]*MetaAgent
	observerAgentMap map[uuid.UUID]*ObserverAgent

	stateMemory         []map[uuid.UUID][]byte
	environmentMemory   []map[string][]byte
	maxStateMemoryDepth int

	environmentVariables map[string][]byte

	metaHierarchy MetaHierarchy

	maxDuration time.Duration

	areInternalMessagesSynchronous bool

	// Package Exposure
	OnUpdateEnvironment Event[*Server]
	OnIterationFinished Event[*Server]
}

func CreateServer(numModelAgents []int, createModelAgents []func(*Server) IGenericAgent,
	numObserverAgents []int, createObserverAgents []func(*Server) IGenericAgent,
	stateMemoryDepths int, iterations int, maxDuration time.Duration, agentBandwidth int) *Server {
	modelCapacity := 0
	for _, num := range numModelAgents {
		modelCapacity += num
	}
	observerCapacity := 0
	for _, num := range numObserverAgents {
		observerCapacity += num
	}

	serv := &Server{
		BaseServer:           server.CreateBaseServer[IGenericAgent](iterations, 4, maxDuration, agentBandwidth),
		modelAgents:          make([]uuid.UUID, 0, modelCapacity),
		observerAgents:       make([]uuid.UUID, 0, observerCapacity),
		metaAgents:           make([]uuid.UUID, 0),
		modelAgentMap:        make(map[uuid.UUID]*ModelAgent, modelCapacity),
		observerAgentMap:     make(map[uuid.UUID]*ObserverAgent, observerCapacity),
		metaAgentMap:         make(map[uuid.UUID]*MetaAgent),
		maxStateMemoryDepth:  stateMemoryDepths,
		stateMemory:          make([]map[uuid.UUID][]byte, 0, stateMemoryDepths),
		environmentMemory:    make([]map[string][]byte, 0, stateMemoryDepths),
		environmentVariables: make(map[string][]byte),
	}

	for i, num := range numObserverAgents {
		for j := 0; j < num; j++ {
			serv.AddAgent(createObserverAgents[i](serv))
		}
	}
	for i, num := range numModelAgents {
		for j := 0; j < num; j++ {
			serv.AddAgent(createModelAgents[i](serv))
		}
	}

	serv.areInternalMessagesSynchronous = false

	serv.metaHierarchy.createMetaHierarchy(serv.modelAgents)
	serv.maxDuration = maxDuration
	serv.SetGameRunner(serv)
	return serv
}

// Internal running functions (partially exposed due to base package)

func (serv *Server) RunTurn(iteration, turn int) {
	fmt.Printf("Running iteration %v, turn %v\n", iteration+1, turn+1)

	// Communication Partner Search
	if turn == 0 {
		for _, ag := range serv.GetAgentMap() {
			ag.setupCommunicationPartnerSearch()
		}
		for _, ag := range serv.GetAgentMap() {
			ag.handleCommunicationPartnerSearch()
		}
	}

	// Main Communication Phase
	if turn == 1 {
		for _, ag := range serv.GetAgentMap() {
			ag.setupMainCommunicationPhase()
		}
		for _, ag := range serv.GetAgentMap() {
			ag.handleMainCommunicationPhase()
		}
	}

	// State update
	if turn == 2 {
		for _, ag := range serv.GetAgentMap() {
			ag.setupStateUpdatePhase()
		}
		for _, ag := range serv.GetAgentMap() {
			ag.handleStateUpdatePhase()
		}
	}

	// Cleanup
	if turn == 3 {
		serv.cleanupMetaAgents()
		serv.saveStatesToMemory()
		serv.OnUpdateEnvironment.invoke(serv)
		serv.OnIterationFinished.invoke(serv)
		for _, ag := range serv.GetAgentMap() {
			ag.SignalMessagingComplete()
		}
	}
}

func (serv *Server) cleanupMetaAgents() {
	scheduledForDissolve := make([]*MetaAgent, 0, len(serv.metaAgents))
	for _, id := range serv.metaAgents {
		ag := serv.metaAgentMap[id]
		if ag.hasDissolved {
			scheduledForDissolve = append(scheduledForDissolve, ag)
		}
	}

	for _, ag := range scheduledForDissolve {
		serv.unsubsumeAgents(ag)
		serv.deleteMetaAgent(ag)
	}
}

func (serv *Server) unsubsumeAgents(ag *MetaAgent) {
	for _, id := range ag.subsumedAgents {
		modelAgent, ok := ag.serv.modelAgentMap[id]
		if ok {
			modelAgent.isSubsumed = false
			modelAgent.subsumedBy = nil
		}
		metaAgent, ok := ag.serv.metaAgentMap[id]
		if ok {
			metaAgent.isSubsumed = false
			metaAgent.subsumedBy = nil
		}
	}
}

func (serv *Server) deleteMetaAgent(ag *MetaAgent) {
	serv.metaAgents = slices.DeleteFunc(serv.metaAgents, func(cmp uuid.UUID) bool { return cmp == ag.GetID() })
	serv.metaHierarchy.dissolve(ag.GetID())
	delete(serv.metaAgentMap, ag.GetID())
	serv.RemoveAgent(ag)
}

func (serv *Server) saveStatesToMemory() {
	if serv.maxStateMemoryDepth == 0 {
		return
	}
	states := make(map[uuid.UUID][]byte)
	for id, ma := range serv.modelAgentMap {
		states[id] = ma.state
	}
	serv.stateMemory = append(serv.stateMemory, states)
	if len(serv.stateMemory) > serv.maxStateMemoryDepth {
		serv.stateMemory = serv.stateMemory[1:]
	}
	serv.environmentMemory = append(serv.environmentMemory, serv.environmentVariables)
	if len(serv.environmentMemory) > serv.maxStateMemoryDepth {
		serv.environmentMemory = serv.environmentMemory[1:]
	}
}

func (serv *Server) RunStartOfIteration(i int) {
	fmt.Printf("Starting iteration %v\n", i+1)
	fmt.Println()
}

func (serv *Server) RunEndOfIteration(i int) {
	fmt.Println()
	fmt.Printf("Ending iteration %v\n", i+1)
}

// Exposed Getters/Setters

func (serv *Server) GetStateMemory() []map[uuid.UUID][]byte {
	return serv.stateMemory
}

func (serv *Server) GetObserverAgentMap() map[uuid.UUID]*ObserverAgent {
	return serv.observerAgentMap
}

func (serv *Server) GetMetaAgentMap() map[uuid.UUID]*MetaAgent {
	return serv.metaAgentMap
}

func (serv *Server) GetModelAgentMap() map[uuid.UUID]*ModelAgent {
	return serv.modelAgentMap
}

func (serv *Server) GetMetaAgents() []uuid.UUID {
	return serv.metaAgents
}

func (serv *Server) GetObserverAgents() []uuid.UUID {
	return serv.observerAgents
}

func (serv *Server) GetModelAgents() []uuid.UUID {
	return serv.modelAgents
}

func (serv *Server) GetMetaHierarchy() *MetaHierarchy {
	return &serv.metaHierarchy
}

func (serv *Server) GetEnvironmentVariables() map[string][]byte { return serv.environmentVariables }

func (serv *Server) GetEnvironmentVariable(name string) ([]byte, bool) {
	value, ok := serv.environmentVariables[name]
	return value, ok
}

func (serv *Server) SetEnvironmentVariable(key string, value []byte) {
	serv.environmentVariables[key] = value
}

func (serv *Server) SetInternalMessagesSynchronous(value bool) {
	serv.areInternalMessagesSynchronous = value
}

// Rollback and Resim functionality below are in a fully untested and unfinished draft state.
// These are NOT part of the core functionality of the framework and should be handled with care!

var printedRollbackWarning = false
var printedResimWarning = false

func (serv *Server) RollbackState(iterationsAgo int) {
	if !printedRollbackWarning {
		fmt.Printf("WARNING! Rollback functionality is in early alpha state and may not work as intended. Please report problems on the github repository!\n")
		printedRollbackWarning = true
	}

	if iterationsAgo < 1 {
		fmt.Printf("Rollback iterations cannot be 0! Was chosen as (%v)\n", iterationsAgo)
		return
	}
	if iterationsAgo > len(serv.stateMemory) {
		fmt.Printf("Rollback iterations (%v) exceeds state memory depth (%v)\n", iterationsAgo, len(serv.stateMemory))
		return
	}

	fmt.Printf("Rolling back %v iterations.\n", iterationsAgo)

	backupState := serv.stateMemory[len(serv.stateMemory)-iterationsAgo]
	for id, agent := range serv.modelAgentMap {
		agent.state = backupState[id]
	}
	serv.environmentVariables = serv.environmentMemory[len(serv.stateMemory)-iterationsAgo]

	serv.stateMemory = serv.stateMemory[:len(serv.stateMemory)-iterationsAgo]
	serv.environmentMemory = serv.environmentMemory[:len(serv.stateMemory)-iterationsAgo]

	// reconstruct meta hierarchy?
}

func (serv *Server) Resimulate(iterations int, agents []uuid.UUID) map[uuid.UUID][]byte {
	if !printedResimWarning {
		fmt.Printf("WARNING! Resimulation functionality is in early alpha state and may not work as intended. Please report problems on the github repository!\n")
		printedResimWarning = true
	}

	if iterations < 1 {
		fmt.Printf("Resim iterations cannot be 0! Was chosen as (%v)\n", iterations)
		return nil
	}
	if iterations > len(serv.stateMemory) {
		fmt.Printf("Resim iterations (%v) exceeds state memory depth (%v)\n", iterations, len(serv.stateMemory))
		return nil
	}

	// Create Local Resim Server, load agents & environment onto it
	resimServ := CreateServer(make([]int, 0), make([]func(*Server) IGenericAgent, 0),
		make([]int, 0), make([]func(*Server) IGenericAgent, 0),
		serv.maxStateMemoryDepth, iterations, serv.maxDuration, serv.GetAgentMessagingBandwidth())

	for _, id := range agents {
		resimServ.AddAgent(serv.GetAgentMap()[id])
	}

	resimServ.environmentVariables = serv.environmentMemory[len(serv.stateMemory)-iterations]
	resimServ.stateMemory = serv.stateMemory[:len(serv.stateMemory)-iterations]
	resimServ.environmentMemory = serv.environmentMemory[:len(serv.stateMemory)-iterations]

	// Backup current agent state & modify agents for resim
	subsumedMap := make(map[uuid.UUID]*MetaAgent)
	for id := range resimServ.GetAgentMap() {
		modelAgent, ok := serv.modelAgentMap[id]
		if ok {
			resimServ.modelAgents = append(resimServ.modelAgents, id)
			resimServ.modelAgentMap[id] = modelAgent

			modelAgent.modelAgents = &resimServ.modelAgents
			modelAgent.observerAgents = &resimServ.observerAgents
			modelAgent.environment = &resimServ.environmentVariables
			modelAgent.state = resimServ.stateMemory[len(resimServ.stateMemory)-1][id]

			subsumedMap[id] = modelAgent.subsumedBy
			if modelAgent.isSubsumed && !slices.Contains(agents, modelAgent.subsumedBy.GetID()) {
				modelAgent.isSubsumed = false
				modelAgent.subsumedBy = nil
			}
		}

		metaAgent, ok := serv.metaAgentMap[id]
		if ok {
			resimServ.metaAgents = append(resimServ.metaAgents, id)
			resimServ.metaAgentMap[id] = metaAgent

			metaAgent.serv = resimServ
			metaAgent.state.applyStateChange(resimServ.stateMemory[len(resimServ.stateMemory)-1])

			subsumedMap[id] = metaAgent.subsumedBy
			if metaAgent.isSubsumed && !slices.Contains(agents, metaAgent.subsumedBy.GetID()) {
				metaAgent.isSubsumed = false
				metaAgent.subsumedBy = nil
			}
		}

		observerAgent, ok := serv.observerAgentMap[id]
		if ok {
			resimServ.observerAgents = append(resimServ.observerAgents, id)
			resimServ.observerAgentMap[id] = observerAgent

			observerAgent.modelAgents = &resimServ.modelAgents
			observerAgent.metaAgents = &resimServ.metaAgents
			observerAgent.observedModelAgents, observerAgent.observedMetaAgents = observerAgent.observationStrategy()
			observerAgent.serv = resimServ
		}
	}
	resimServ.metaHierarchy.createMetaHierarchy(resimServ.modelAgents)
	for _, ma := range resimServ.metaAgentMap {
		resimServ.metaHierarchy.subsume(ma.GetID(), ma.subsumedAgents)
	}

	// Perform resimulation
	for i := range iterations {
		for j := range 4 {
			resimServ.RunTurn(i, j)
		}
	}
	resultState := resimServ.stateMemory[len(resimServ.stateMemory)-1]

	// Reset agents back to original
	for id := range resimServ.GetAgentMap() {
		modelAgent, ok := serv.modelAgentMap[id]
		if ok {
			modelAgent.modelAgents = &serv.modelAgents
			modelAgent.observerAgents = &serv.observerAgents
			modelAgent.environment = &serv.environmentVariables
			modelAgent.state = serv.stateMemory[len(serv.stateMemory)-1][id]

			subsumedBy, ok := subsumedMap[id]
			if ok && subsumedBy != nil {
				modelAgent.isSubsumed = true
				modelAgent.subsumedBy = subsumedBy
			}
		}

		metaAgent, ok := serv.metaAgentMap[id]
		if ok {
			metaAgent.serv = serv
			metaAgent.state.applyStateChange(serv.stateMemory[len(serv.stateMemory)-1])

			subsumedBy, ok := subsumedMap[id]
			if ok && subsumedBy != nil {
				metaAgent.isSubsumed = true
				metaAgent.subsumedBy = subsumedBy
			}
		}

		observerAgent, ok := serv.observerAgentMap[id]
		if ok {
			observerAgent.modelAgents = &serv.modelAgents
			observerAgent.metaAgents = &serv.metaAgents
			observerAgent.observedModelAgents, observerAgent.observedMetaAgents = observerAgent.observationStrategy()
			observerAgent.serv = serv
		}
	}

	return resultState
}
