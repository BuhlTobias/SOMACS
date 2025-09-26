package main

import (
	"GRL1/SOMACS"
	"fmt"
	"github.com/google/uuid"
	"math/rand"
	"os"
	"time"
)

// Basic Example of a scenario model and scenario observer agent & server instantiation and access

// Model Agent

const MSGTYPE_HELLO = 1
const MSGTYPE_WORLD = 2

type HelloModelAgent struct {
	*SOMACS.ModelAgent
	Cluster byte

	ExpectedWorldMessages int
	ReceivedWorldMessages int
	SentWorldMessages     int
}

func CreateHelloModelAgent(serv *SOMACS.Server) SOMACS.IGenericAgent {
	hma := &HelloModelAgent{ModelAgent: SOMACS.CreateModelAgentBase(serv)}
	hma.Cluster = byte(rand.Intn(numClusters))

	// For (1) Communication Partner Search
	setupCommunicationPartnerSearch := func(*SOMACS.ModelAgent) {
		shuffleTimer, _ := hma.GetEnvironmentVariable("ShuffleTimer")
		if shuffleTimer[0] > 0 {
			return
		}
		hma.Cluster = byte(rand.Intn(numClusters))
		hma.SetValidationRequestData(append(make([]byte, 0, 1), hma.Cluster))
	}
	hma.OnSetupCommunicationPartnerSearch.Subscribe(&setupCommunicationPartnerSearch)
	hma.SetValidationFunc(
		func(msg SOMACS.Message) bool {
			if len(msg.Data) == 0 {
				panic("HelloModelAgent Validation Request without Data found - this message should always contain the cluster!")
			}
			return msg.Data[0] == hma.Cluster
		},
	)
	hma.SetValidationRequestData(append(make([]byte, 0, 1), hma.Cluster))

	// For (2) Main Communication Phase
	setupMainCommunicationPhase := func(*SOMACS.ModelAgent) {
		hma.ExpectedWorldMessages = len(hma.GetValidCommunicationPartners())
		hma.ReceivedWorldMessages = 0
		hma.SentWorldMessages = 0
	}
	hma.OnSetupMainCommunicationPhase.Subscribe(&setupMainCommunicationPhase)
	beginMainCommunicationPhase := func(*SOMACS.ModelAgent) {
		msg := hma.CreateHelloMessage()
		if isExampleSynchronous {
			hma.BroadcastSynchronousMessageToRecipients(msg, hma.GetValidCommunicationPartners())
		} else {
			hma.BroadcastMessageToRecipients(msg, hma.GetValidCommunicationPartners())
		}
	}
	hma.OnBeginMainCommunicationPhase.Subscribe(&beginMainCommunicationPhase)

	handleMessage := func(msg SOMACS.Message) {
		switch msg.MessageType {
		default:
			break
		case MSGTYPE_HELLO:
			hma.HandleHelloMessage(msg)
			break
		case MSGTYPE_WORLD:
			hma.HandleWorldMessage(msg)
			break
		}
	}
	hma.OnHandleMessage.Subscribe(&handleMessage)

	// For (3) State Update Phase
	hma.SetStateUpdateFunc(
		func() []byte {
			state := make([]byte, 0, 1)
			state = append(state, byte(hma.ReceivedWorldMessages))
			return state
		},
	)

	return hma
}

// Main Communication Phase Protocol

func (hma *HelloModelAgent) CreateHelloMessage() *SOMACS.Message {
	msg := hma.CreateMessage()
	msg.MessageType = MSGTYPE_HELLO
	return msg
}

func (hma *HelloModelAgent) HandleHelloMessage(msg SOMACS.Message) {
	//fmt.Printf("%s hears %s say \"Hello!\"\n", hma.GetID(), msg.GetSender())
	response := hma.CreateWorldMessage()
	response.Recipient = msg.GetSender()
	hma.SentWorldMessages++
	if isExampleSynchronous {
		hma.SendSynchronousMessage(response, msg.GetSender())
	} else {
		hma.SendMessage(response, msg.GetSender())
	}
	hma.CheckFinishedMainMessaging()
}

func (hma *HelloModelAgent) CreateWorldMessage() *SOMACS.Message {
	msg := hma.CreateMessage()
	msg.MessageType = MSGTYPE_WORLD
	return msg
}

func (hma *HelloModelAgent) HandleWorldMessage(SOMACS.Message) {
	//fmt.Printf("%s hears %s respond \"World!\"\n", hma.GetID(), msg.GetSender())
	hma.ReceivedWorldMessages++
	hma.CheckFinishedMainMessaging()
}

func (hma *HelloModelAgent) CheckFinishedMainMessaging() {
	if hma.ReceivedWorldMessages >= hma.ExpectedWorldMessages && hma.SentWorldMessages >= hma.ExpectedWorldMessages {
		hma.EndMainCommunicationPhase()
	}
}

// Observer Agent

type HelloObserverAgent struct {
	*SOMACS.ObserverAgent
}

func CreateHelloObserverAgent(serv *SOMACS.Server) SOMACS.IGenericAgent {
	hoa := &HelloObserverAgent{ObserverAgent: SOMACS.CreateObserverAgentBase(serv)}

	hoa.SetObservationStrategy(
		func() (*[]uuid.UUID, *[]uuid.UUID) {
			if len(*hoa.GetObservedModelAgents()) > 0 {
				return hoa.GetObservedModelAgents(), hoa.GetObservedMetaAgents()
			}
			return hoa.ObserveAllNotSubsumedModelAgents()
		},
	)

	onAfterAllStateUpdatesReceived := hoa.PredictMessagingGroupsAndCreateMetaAgents
	hoa.ObserverAgent.OnAfterAllStateUpdatesReceived.Subscribe(&onAfterAllStateUpdatesReceived)

	return hoa
}

func (hoa *HelloObserverAgent) PredictMessagingGroupsAndCreateMetaAgents(statistics *SOMACS.ObserverStatistics) {
	// Pattern Detection/Classification
	predictedGroups := make([][]uuid.UUID, 0, len(*hoa.GetObservedModelAgents()))
	statistics.StateStatistics.ClearEmptyStates()
	for _, agent := range *hoa.GetObservedModelAgents() {
		fitsInGroup := false
		for i, group := range predictedGroups {
			groupNotEmpty := group != nil && len(group) > 0
			if !groupNotEmpty {
				continue
			}
			isCommunicationMatching := statistics.MessageStatistics.HasCommunicatedWithAll(agent, group)
			if !isCommunicationMatching {
				continue
			}
			predictedGroups[i] = append(group, agent)
			fitsInGroup = true
		}
		if !fitsInGroup {
			newGroup := make([]uuid.UUID, 0, len(*hoa.GetObservedModelAgents()))
			newGroup = append(newGroup, agent)
			predictedGroups = append(predictedGroups, newGroup)
		}
	}

	// Scheduling of Meta Agents according to predicted groups
	minClusterSize := 5
	for _, group := range predictedGroups {
		if len(group) < minClusterSize {
			continue
		}
		fmt.Printf("Creating a meta agent consisting of (%v) agents...\n", len(group))
		maGroup := make([]*SOMACS.ModelAgent, 0, len(group))
		for _, ag := range group {
			maGroup = append(maGroup, hoa.GetServer().GetModelAgentMap()[ag])
		}

		// Define partner search, predict, verify, evaluate (optional), explain (optional)
		partnerSearch := func(messageStatistics *SOMACS.MessageStatistics, state *SOMACS.MetaState) (map[uuid.UUID]map[uuid.UUID]bool, map[uuid.UUID][]uuid.UUID) {
			responses := make(map[uuid.UUID]map[uuid.UUID]bool)
			internal := make(map[uuid.UUID][]uuid.UUID)

			subsumedModelAgents := make([]uuid.UUID, 0)
			for ag := range state.ModelStates {
				subsumedModelAgents = append(subsumedModelAgents, ag)
			}

			for requester := range messageStatistics.GetCommunicationMap() {
				msgs, _ := messageStatistics.GetAllMessagesFromAgent(requester)
				valid := maGroup[0].ValidateMessage(msgs[0])
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

			return responses, internal
		}

		predict := func(messageStatistics *SOMACS.MessageStatistics, state *SOMACS.MetaState) map[uuid.UUID][]byte {
			var ag0 uuid.UUID
			for ag := range state.ModelStates {
				ag0 = ag
				break
			}

			worldMessages, _ := messageStatistics.GetMessagesOfTypeToAgent(ag0, MSGTYPE_WORLD)
			predictedState := make([]byte, 0, 1)
			predictedState = append(predictedState, byte(len(worldMessages)))

			ret := make(map[uuid.UUID][]byte)
			for ag := range state.ModelStates {
				ret[ag] = predictedState
			}
			return ret
		}

		errorTolerance := float32(0.25)
		verify := func(messageStatistics *SOMACS.MessageStatistics, state *SOMACS.MetaState, baseState map[uuid.UUID][]byte) bool {
			for id, predictedState := range state.ModelStates {
				currentValue := float32(predictedState[0])
				baseValue := float32(baseState[id][0])
				if baseValue <= 1 {
					continue
				}
				errorValue := max((currentValue+1.0)/(baseValue+1.0), (baseValue+1.0)/(currentValue+1.0)) - 1.0
				if errorValue > errorTolerance {
					fmt.Printf("Meta agent consisting of (%v) agents fails verify with error of (%v)\n", len(state.ModelStates), errorValue)
					return false
				}
			}

			shuffleTimer, _ := hoa.GetServer().GetEnvironmentVariable("ShuffleTimer")
			if shuffleTimer[0] <= 1 {
				fmt.Printf("Meta agent consisting of (%v) agents dissolves due to scheduled shuffle\n", len(state.ModelStates))
				return false
			}

			return true
		}

		evaluate := func(ag *SOMACS.MetaAgent) float32 {
			subsumedAgents := *ag.GetSubsumedAgents()
			reportedValue := ag.GetState().ModelStates[subsumedAgents[0]][0]
			sizeSuggestValue := len(subsumedAgents) - 1
			sizeBasedAccuracy := min(float32(reportedValue)/float32(sizeSuggestValue), float32(sizeSuggestValue)/float32(reportedValue))

			if evaluateVerbose {
				fmt.Printf("Meta Agent (%v) of size %v reports value (%v). This suggests accuracy (%v).\n",
					ag.GetID(), len(subsumedAgents), reportedValue, sizeBasedAccuracy)
			}
			accuracyList = append(accuracyList, sizeBasedAccuracy)
			return sizeBasedAccuracy
		}

		explain := func(ag *SOMACS.MetaAgent) {
			if !explainabilityVerbose {
				return
			}
			fmt.Printf("Explanation for Meta Agent (%v):\n"+
				"\tPredicts a cluster of (%v) agents around uuid (%v).\n"+
				"\tAgents in the cluster received (%v) \"World\" messages at base, and (%v) last iteration.\n"+
				"\tIf the uniform predicted value is within (%v)%s of the measured base value, it passes verification.\n",
				ag.GetID(),
				len(*ag.GetSubsumedAgents()), (*ag.GetSubsumedAgents())[0],
				ag.GetCondition().GetBaseState()[(*ag.GetSubsumedAgents())[0]][0], ag.GetState().ModelStates[(*ag.GetSubsumedAgents())[0]][0],
				int(errorTolerance*100), "%")
			fmt.Printf("Counterfactual Explanation for Meta Agent (%v):\n"+
				"\tMeta agent would fail verification if prime agent received more than (%v) or less than (%v) messages\n"+
				"\tMeta agent would fail verification if the \"Time until clusters shuffle\" environment variable was 1 or lower.\n",
				ag.GetID(),
				int((1.0+errorTolerance)*float32(ag.GetCondition().GetBaseState()[(*ag.GetSubsumedAgents())[0]][0])),
				int(1.0/(1.0+errorTolerance)*float32(ag.GetCondition().GetBaseState()[(*ag.GetSubsumedAgents())[0]][0])))
		}

		hoa.ScheduleMetaAgent(maGroup, make([]*SOMACS.MetaAgent, 0), partnerSearch, predict, verify, evaluate, explain)
	}
}

// Server

func CreateHelloServer(numAgents, iterations int, maxDuration time.Duration, agentBandwidth int) *SOMACS.Server {
	serv := SOMACS.CreateServer([]int{numAgents}, []func(*SOMACS.Server) SOMACS.IGenericAgent{CreateHelloModelAgent},
		[]int{spawnObservers}, []func(*SOMACS.Server) SOMACS.IGenericAgent{CreateHelloObserverAgent},
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
		if printHierarchy {
			fmt.Printf("Meta Hierarchy:\n")
			fmt.Printf(serv.GetMetaHierarchy().ToStringCompact())
		}
	}
	serv.OnIterationFinished.Subscribe(&onIterationFinished)
	serv.SetInternalMessagesSynchronous(isExampleSynchronous)

	return serv
}

// Global variables used for testing
var numClusters = 3
var spawnObservers = 1
var evaluateVerbose = true
var explainabilityVerbose = false
var printHierarchy = true
var timeBetweenShuffles = byte(5)
var isExampleSynchronous = true

func CreateExampleSim() {
	serv := CreateHelloServer(200, 10, 100*time.Millisecond, 1000)
	serv.ReportMessagingDiagnostics()
	serv.Start()
}

func PressEnterToExit() {
	fmt.Println("Press [Enter] to exit...")

	var b = make([]byte, 1)
	for {
		_, err := os.Stdin.Read(b)
		if err != nil {
			continue
		}
		return
	}
}

func main() {
	CreateExtendedExampleSim()
	PressEnterToExit()
}
