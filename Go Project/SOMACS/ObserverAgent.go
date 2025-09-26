package SOMACS

import (
	"fmt"
	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/agent"
	"github.com/google/uuid"
	"slices"
)

type ObserverStatistics struct {
	MessageStatistics MessageStatistics
	StateStatistics   StateStatistics
}

type ObserverAgent struct {
	*agent.BaseAgent[IGenericAgent]
	modelAgents *[]uuid.UUID
	metaAgents  *[]uuid.UUID

	// Observed Agents
	observationStrategy func() (*[]uuid.UUID, *[]uuid.UUID)
	observedModelAgents *[]uuid.UUID
	observedMetaAgents  *[]uuid.UUID

	statistics ObserverStatistics

	// For (2) Main Communication Phase
	expectedComMainEnd int
	receivedComMainEnd int

	// For (3) State Update Phase
	expectedStateUpdate int
	receivedStateUpdate int

	// For Meta Agent Creation
	serv                              *Server
	scheduledMetaAgentModelAgents     [][]*ModelAgent
	scheduledMetaAgentMetaAgents      [][]*MetaAgent
	schedulesMetaAgentPartnerSearches []func(*MessageStatistics, *MetaState) (map[uuid.UUID]map[uuid.UUID]bool, map[uuid.UUID][]uuid.UUID)
	scheduledMetaAgentPredicts        []func(*MessageStatistics, *MetaState) map[uuid.UUID][]byte
	scheduledMetaAgentVerifies        []func(*MessageStatistics, *MetaState, map[uuid.UUID][]byte) bool
	scheduledMetaAgentEvaluates       []func(*MetaAgent) float32
	scheduledMetaAgentExplains        []func(*MetaAgent)

	// Package Exposure
	OnHandleMessage                   Event[Message]
	OnSetupCommunicationPartnerSearch Event[*ObserverAgent]
	OnSetupMainCommunicationPhase     Event[*ObserverAgent]
	OnSetupStateUpdatePhase           Event[*ObserverAgent]

	OnModelStateUpdateReceived     Event[Message]
	OnMetaStateUpdateReceived      Event[Message]
	OnAfterAllStateUpdatesReceived Event[*ObserverStatistics]
}

func CreateObserverAgentBase(serv *Server) *ObserverAgent {
	oa := &ObserverAgent{BaseAgent: agent.CreateBaseAgent(serv)}
	oa.modelAgents = &serv.modelAgents
	oa.metaAgents = &serv.metaAgents

	oa.observationStrategy = oa.ObserveAllNotSubsumedModelAgents
	observedModelAgents := make([]uuid.UUID, 0, len(*oa.modelAgents))
	oa.observedModelAgents = &observedModelAgents
	observedMetaAgents := make([]uuid.UUID, 0)
	oa.observedMetaAgents = &observedMetaAgents

	oa.statistics.MessageStatistics.createMessageStatistics()
	oa.statistics.StateStatistics.createStateStatistics()

	oa.serv = serv
	oa.scheduledMetaAgentModelAgents = make([][]*ModelAgent, 0)
	oa.schedulesMetaAgentPartnerSearches = make([]func(*MessageStatistics, *MetaState) (map[uuid.UUID]map[uuid.UUID]bool, map[uuid.UUID][]uuid.UUID), 0)
	oa.scheduledMetaAgentPredicts = make([]func(*MessageStatistics, *MetaState) map[uuid.UUID][]byte, 0)
	oa.scheduledMetaAgentVerifies = make([]func(*MessageStatistics, *MetaState, map[uuid.UUID][]byte) bool, 0)
	oa.scheduledMetaAgentEvaluates = make([]func(*MetaAgent) float32, 0)
	oa.scheduledMetaAgentExplains = make([]func(*MetaAgent), 0)

	serv.observerAgents = append(serv.observerAgents, oa.GetID())
	serv.observerAgentMap[oa.GetID()] = oa
	return oa
}

func createObserverAgent(serv *Server) IGenericAgent {
	return CreateObserverAgentBase(serv)
}

func (oa *ObserverAgent) handleMessage(msg Message) {
	if slices.Contains(*oa.observedModelAgents, msg.Recipient) || slices.Contains(*oa.observedModelAgents, msg.GetSender()) {
		oa.statistics.MessageStatistics.recordMessage(msg)
	}
	if !slices.Contains(*oa.observedModelAgents, msg.GetSender()) && !slices.Contains(*oa.observedMetaAgents, msg.GetSender()) {
		return
	}
	switch msg.MessageType {
	case MSGTYPE_COM_MAIN_END:
		oa.handleMainPhaseEndMessage(msg)
		return
	case MSGTYPE_COM_STATE_UPDATE:
		oa.handleStateUpdateMessage(msg)
		return
	case MSGTYPE_META_STATE_UPDATE:
		oa.handleMetaStateUpdateMessage(msg)
		return
	}
	oa.OnHandleMessage.invoke(msg)
}

func (oa *ObserverAgent) setupCommunicationPartnerSearch() {
	oa.OnSetupCommunicationPartnerSearch.invoke(oa)
}

func (oa *ObserverAgent) handleCommunicationPartnerSearch() {
	oa.SignalMessagingComplete()
}

// For (2) Main Communication

func (oa *ObserverAgent) setupMainCommunicationPhase() {
	oa.OnSetupMainCommunicationPhase.invoke(oa)
	observedModelAgents, observedMetaAgents := oa.observationStrategy()
	if observedModelAgents == nil {
		if oa.observedModelAgents != nil {
			*oa.observedModelAgents = (*oa.observedModelAgents)[:0]
			observedModelAgents = oa.observedModelAgents
		}
		oma := make([]uuid.UUID, 0)
		observedModelAgents = &oma
	}
	oa.observedModelAgents = observedModelAgents

	if observedMetaAgents == nil {
		if oa.observedMetaAgents != nil {
			*oa.observedMetaAgents = (*oa.observedMetaAgents)[:0]
			observedMetaAgents = oa.observedMetaAgents
		}
		oma := make([]uuid.UUID, 0)
		observedMetaAgents = &oma
	}
	oa.observedMetaAgents = observedMetaAgents

	if len(*oa.observedModelAgents) > 0 {
		fmt.Printf("Observer agent (%v) observes (%v) model agents\n", oa.GetID(), len(*oa.observedModelAgents))
	}
	if len(*oa.observedMetaAgents) > 0 {
		fmt.Printf("Observer agent (%v) observes (%v) meta agents\n", oa.GetID(), len(*oa.observedMetaAgents))
	}

	oa.expectedComMainEnd = len(*oa.observedModelAgents)
	oa.receivedComMainEnd = 0
	oa.statistics.MessageStatistics.clear()
}

func (oa *ObserverAgent) handleMainCommunicationPhase() {
}

func (oa *ObserverAgent) handleMainPhaseEndMessage(msg Message) {
	oa.statistics.MessageStatistics.recordSignaledMainMessagingComplete(msg.GetSender())
	oa.receivedComMainEnd++
	if oa.receivedComMainEnd == oa.expectedComMainEnd {
		oa.SignalMessagingComplete()
	}
}

// For (3) State Update

func (oa *ObserverAgent) setupStateUpdatePhase() {
	oa.OnSetupStateUpdatePhase.invoke(oa)
	oa.statistics.StateStatistics.clear()
	oa.expectedStateUpdate = len(*oa.observedModelAgents) + len(*oa.observedMetaAgents)
	oa.receivedStateUpdate = 0
}

func (oa *ObserverAgent) handleStateUpdatePhase() {}

func (oa *ObserverAgent) handleStateUpdateMessage(msg Message) {
	oa.OnModelStateUpdateReceived.invoke(msg)
	oa.statistics.StateStatistics.recordState(msg)
	oa.receivedStateUpdate++
	oa.checkAllStateUpdatesReceived()
}

func (oa *ObserverAgent) handleMetaStateUpdateMessage(msg Message) {
	oa.OnMetaStateUpdateReceived.invoke(msg)
	sender := oa.serv.metaAgentMap[msg.GetSender()]
	oa.statistics.StateStatistics.recordMeta(msg.GetSender(), &sender.state, sender.hasDissolved)
	oa.receivedStateUpdate++
	oa.checkAllStateUpdatesReceived()
}

func (oa *ObserverAgent) checkAllStateUpdatesReceived() {
	if oa.receivedStateUpdate == oa.expectedStateUpdate {
		oa.OnAfterAllStateUpdatesReceived.invoke(&oa.statistics)
		oa.createMetaAgents()
		oa.SignalMessagingComplete()
	}
}

func (oa *ObserverAgent) createMetaAgents() {
	for i := range oa.scheduledMetaAgentModelAgents {
		oa.serv.AddAgent(createMetaAgent(oa.serv,
			oa.scheduledMetaAgentModelAgents[i], oa.scheduledMetaAgentMetaAgents[i],
			oa.schedulesMetaAgentPartnerSearches[i], oa.scheduledMetaAgentPredicts[i], oa.scheduledMetaAgentVerifies[i],
			oa.scheduledMetaAgentEvaluates[i], oa.scheduledMetaAgentExplains[i]))
		for _, ag := range oa.scheduledMetaAgentModelAgents[i] {
			l := len(*oa.observedModelAgents)
			for j := 1; j <= l; j++ {
				if (*oa.observedModelAgents)[l-j] == ag.GetID() {
					*oa.observedModelAgents = append((*oa.observedModelAgents)[:l-j], (*oa.observedModelAgents)[l-j+1:]...)
				}
			}
		}
	}
	oa.scheduledMetaAgentModelAgents = oa.scheduledMetaAgentModelAgents[:0]
	oa.scheduledMetaAgentMetaAgents = oa.scheduledMetaAgentMetaAgents[:0]
	oa.schedulesMetaAgentPartnerSearches = oa.schedulesMetaAgentPartnerSearches[:0]
	oa.scheduledMetaAgentPredicts = oa.scheduledMetaAgentPredicts[:0]
	oa.scheduledMetaAgentVerifies = oa.scheduledMetaAgentVerifies[:0]
	oa.scheduledMetaAgentEvaluates = oa.scheduledMetaAgentEvaluates[:0]
	oa.scheduledMetaAgentExplains = oa.scheduledMetaAgentExplains[:0]
}

// Exposed Functions

func (oa *ObserverAgent) CreateMessage() *Message {
	return &Message{BaseMessage: oa.CreateBaseMessage()}
}

func (oa *ObserverAgent) ObserveAllNotSubsumedModelAgents() (*[]uuid.UUID, *[]uuid.UUID) {
	temp := make([]uuid.UUID, 0, len(*oa.modelAgents))
	for _, ag := range *oa.modelAgents {
		if oa.serv.modelAgentMap[ag].isSubsumed {
			continue
		}
		temp = append(temp, ag)
	}
	return &temp, oa.observedMetaAgents
} // Default Strategy

func (oa *ObserverAgent) GetObservedModelAgents() *[]uuid.UUID {
	return oa.observedModelAgents
}

func (oa *ObserverAgent) GetObservedMetaAgents() *[]uuid.UUID {
	return oa.observedMetaAgents
}

func (oa *ObserverAgent) GetServer() *Server {
	return oa.serv
}

func (oa *ObserverAgent) GetStatistics() *ObserverStatistics {
	return &oa.statistics
}

func (oa *ObserverAgent) ScheduleMetaAgent(modelAgents []*ModelAgent, metaAgents []*MetaAgent,
	partnerSearch func(*MessageStatistics, *MetaState) (map[uuid.UUID]map[uuid.UUID]bool, map[uuid.UUID][]uuid.UUID),
	predictState func(*MessageStatistics, *MetaState) map[uuid.UUID][]byte,
	verify func(*MessageStatistics, *MetaState, map[uuid.UUID][]byte) bool,
	evaluate func(*MetaAgent) float32,
	explain func(*MetaAgent)) {
	oa.scheduledMetaAgentModelAgents = append(oa.scheduledMetaAgentModelAgents, modelAgents)
	oa.scheduledMetaAgentMetaAgents = append(oa.scheduledMetaAgentMetaAgents, metaAgents)
	oa.schedulesMetaAgentPartnerSearches = append(oa.schedulesMetaAgentPartnerSearches, partnerSearch)
	oa.scheduledMetaAgentPredicts = append(oa.scheduledMetaAgentPredicts, predictState)
	oa.scheduledMetaAgentVerifies = append(oa.scheduledMetaAgentVerifies, verify)
	oa.scheduledMetaAgentEvaluates = append(oa.scheduledMetaAgentEvaluates, evaluate)
	oa.scheduledMetaAgentExplains = append(oa.scheduledMetaAgentExplains, explain)
}

func (oa *ObserverAgent) SetObservationStrategy(strategy func() (*[]uuid.UUID, *[]uuid.UUID)) {
	oa.observationStrategy = strategy
}
