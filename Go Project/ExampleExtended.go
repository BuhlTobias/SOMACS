package main

import (
	"GRL1/SOMACS"
	"fmt"
	"github.com/google/uuid"
	"time"
)

// Extension to showcase the Iteration II Observer Agent, s. paper
// Subsumes multiple clusters to make partner search faster

type HelloMetaObserverAgent struct {
	*SOMACS.ObserverAgent
}

func CreateHelloMetaObserverAgent(serv *SOMACS.Server) SOMACS.IGenericAgent {
	hoa := &HelloMetaObserverAgent{ObserverAgent: SOMACS.CreateObserverAgentBase(serv)}

	hoa.SetObservationStrategy(
		func() (*[]uuid.UUID, *[]uuid.UUID) {
			metaAgents := make([]uuid.UUID, len(hoa.GetServer().GetMetaAgents()))
			copy(metaAgents, serv.GetMetaAgents())
			return nil, &metaAgents
		},
	)

	onAfterAllStateUpdatesReceived := hoa.CreateMetaAgent
	hoa.ObserverAgent.OnAfterAllStateUpdatesReceived.Subscribe(&onAfterAllStateUpdatesReceived)

	return hoa
}

func (hoa *HelloMetaObserverAgent) CreateMetaAgent(statistics *SOMACS.ObserverStatistics) {
	fmt.Printf("Investigating (%v) meta agents...", *hoa.GetObservedMetaAgents())
	group := *hoa.GetObservedMetaAgents()
	if len(group) < 2 {
		return
	}
	for _, ag := range group {
		if !hoa.GetServer().GetMetaAgentMap()[ag].Verify() {
			return
		}
	}
	fmt.Printf("Creating an iteration II meta agent consisting of (%v) agents...\n", len(group))
	maGroup := make([]*SOMACS.MetaAgent, 0, len(group))
	for _, ag := range group {
		maGroup = append(maGroup, hoa.GetServer().GetMetaAgentMap()[ag])
	}
	partnerSearch := func(messageStatistics *SOMACS.MessageStatistics, state *SOMACS.MetaState) (map[uuid.UUID]map[uuid.UUID]bool, map[uuid.UUID][]uuid.UUID) {
		responses := make(map[uuid.UUID]map[uuid.UUID]bool)
		internal := make(map[uuid.UUID][]uuid.UUID)

		for _, metaState := range state.ChildStates {
			subsumedModelAgents := make([]uuid.UUID, 0)
			for ag := range metaState.ModelStates {
				subsumedModelAgents = append(subsumedModelAgents, ag)
			}

			for requester := range messageStatistics.GetCommunicationMap() {
				msgs, _ := messageStatistics.GetAllMessagesFromAgent(requester)
				valid := hoa.GetServer().GetModelAgentMap()[subsumedModelAgents[0]].ValidateMessage(msgs[0])
				for _, ag := range subsumedModelAgents {
					_, ok := responses[ag]
					if !ok {
						responses[ag] = make(map[uuid.UUID]bool)
					}
					responses[ag][requester] = valid
				}
			}

			for _, ag := range subsumedModelAgents {
				internal[ag] = subsumedModelAgents
			}
		}
		return responses, internal
	}
	predict := func(messageStatistics *SOMACS.MessageStatistics, state *SOMACS.MetaState) map[uuid.UUID][]byte {
		return state.GetModelStatesRecursive()
	}
	verify := func(messageStatistics *SOMACS.MessageStatistics, state *SOMACS.MetaState, baseState map[uuid.UUID][]byte) bool {
		shuffleTimer, _ := hoa.GetServer().GetEnvironmentVariable("ShuffleTimer")
		if shuffleTimer[0] <= 1 {
			return false
		}
		return true
	}
	hoa.ScheduleMetaAgent(make([]*SOMACS.ModelAgent, 0), maGroup, partnerSearch, predict, verify, nil, nil)
}

// Server

func CreateHelloMetaServer(numAgents, iterations int, maxDuration time.Duration, agentBandwidth int) *SOMACS.Server {
	serv := SOMACS.CreateServer([]int{numAgents}, []func(*SOMACS.Server) SOMACS.IGenericAgent{CreateHelloModelAgent},
		[]int{1, 1}, []func(*SOMACS.Server) SOMACS.IGenericAgent{CreateHelloObserverAgent, CreateHelloMetaObserverAgent},
		3, iterations, maxDuration, agentBandwidth)
	serv.SetEnvironmentVariable("ShuffleTimer", []byte{timeBetweenShuffles})
	onUpdateEnvironment := func(serv *SOMACS.Server) {
		timer, _ := serv.GetEnvironmentVariable("ShuffleTimer")
		if timer[0] == 0 {
			serv.SetEnvironmentVariable("ShuffleTimer", []byte{timeBetweenShuffles})
			return
		}
		serv.SetEnvironmentVariable("ShuffleTimer", []byte{timer[0] - 1})
	}
	serv.OnUpdateEnvironment.Subscribe(&onUpdateEnvironment)
	onIterationFinished := func(serv *SOMACS.Server) {
		for _, ma := range serv.GetMetaAgentMap() {
			ma.Evaluate()
		}
		for _, ma := range serv.GetMetaAgentMap() {
			ma.Explain()
		}
		fmt.Printf("Meta Hierarchy:\n")
		fmt.Printf(serv.GetMetaHierarchy().ToStringCompact())
	}
	serv.OnIterationFinished.Subscribe(&onIterationFinished)
	serv.SetInternalMessagesSynchronous(isExampleSynchronous)
	return serv
}

func CreateExtendedExampleSim() {
	serv := CreateHelloMetaServer(200, 10, 100*time.Millisecond, 250)
	serv.ReportMessagingDiagnostics()
	serv.Start()
}

func CreateLargeExtendedExampleSim() {
	numClusters = 5
	serv := CreateHelloMetaServer(500, 10, 100*time.Millisecond, 1000)
	serv.ReportMessagingDiagnostics()
	serv.Start()
}
