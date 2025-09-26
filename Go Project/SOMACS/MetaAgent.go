package SOMACS

import (
	"fmt"
	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/agent"
	"github.com/google/uuid"
	"slices"
)

type MetaAgent struct {
	*agent.BaseAgent[IGenericAgent]

	messageStatistics   MessageStatistics
	subsumedModelAgents []*ModelAgent
	subsumedMetaAgents  []*MetaAgent
	subsumedAgents      []uuid.UUID
	externalModelAgents []uuid.UUID
	state               MetaState

	expectedComValidRequests      int
	receivedComValidRequests      int
	isProcessingPartnerValidation bool

	subsumedAgentsFinishedMainPhase int

	isSubsumed bool
	subsumedBy *MetaAgent

	partnerSearch func(*MessageStatistics, *MetaState) (map[uuid.UUID]map[uuid.UUID]bool, map[uuid.UUID][]uuid.UUID) //ret: responses map[from][to]valid, internal map[agent][]partners
	predict       func(*MessageStatistics, *MetaState) map[uuid.UUID][]byte
	condition     MetaCondition
	hasDissolved  bool

	evaluate func(*MetaAgent) float32

	serv *Server
}

func createMetaAgent(serv *Server, modelAgents []*ModelAgent, metaAgents []*MetaAgent,
	partnerSearch func(*MessageStatistics, *MetaState) (map[uuid.UUID]map[uuid.UUID]bool, map[uuid.UUID][]uuid.UUID),
	predict func(*MessageStatistics, *MetaState) map[uuid.UUID][]byte,
	verify func(*MessageStatistics, *MetaState, map[uuid.UUID][]byte) bool,
	evaluate func(*MetaAgent) float32,
	explain func(*MetaAgent)) IGenericAgent {
	ma := &MetaAgent{BaseAgent: agent.CreateBaseAgent(serv)}

	ma.messageStatistics.createMessageStatistics()
	ma.subsumedModelAgents = modelAgents
	ma.subsumedMetaAgents = metaAgents
	ma.subsumedAgents = make([]uuid.UUID, len(modelAgents)+len(metaAgents))
	subsumedModelAgentStates := make(map[uuid.UUID][]byte)
	for i, ag := range modelAgents {
		subsumedModelAgentStates[ag.GetID()] = ag.state
		ma.subsumedAgents[i] = ag.GetID()
		ag.isSubsumed = true
		ag.subsumedBy = ma
	}
	subsumedMetaAgentStates := make(map[uuid.UUID]*MetaState)
	for i, ag := range metaAgents {
		subsumedMetaAgentStates[ag.GetID()] = &ag.state
		ma.subsumedAgents[i+len(modelAgents)] = ag.GetID()
		ag.isSubsumed = true
		ag.subsumedBy = ma
	}
	ma.state.createMetaState(subsumedModelAgentStates, subsumedMetaAgentStates)
	ma.externalModelAgents = make([]uuid.UUID, 0, len(serv.modelAgents))
	for _, ag := range serv.modelAgents {
		if ma.isSubsumedByMetaTree(ag) {
			continue
		}
		ma.externalModelAgents = append(ma.externalModelAgents, ag)
	}

	ma.isSubsumed = false
	ma.subsumedBy = nil

	ma.partnerSearch = partnerSearch
	ma.predict = predict
	ma.condition.createMetaCondition(verify, ma.state.GetModelStatesRecursive())
	ma.hasDissolved = false

	ma.evaluate = func(ma *MetaAgent) float32 {
		fmt.Printf("No Evaluation Method for Meta Agent provided.\n")
		return 0
	}

	if evaluate != nil {
		ma.SetEvaluateFunc(evaluate)
	}
	if explain != nil {
		ma.SetExplanationFunc(explain)
	}

	ma.serv = serv

	serv.metaAgents = append(serv.metaAgents, ma.GetID())
	serv.metaAgentMap[ma.GetID()] = ma
	serv.metaHierarchy.subsume(ma.GetID(), ma.subsumedAgents)
	return ma
}

func (ma *MetaAgent) handleMessage(msg Message) {
	switch msg.MessageType {
	case MSGTYPE_COM_MAIN_END:
		ma.handleMainPhaseEndMessage()
		break
	case MSGTYPE_COM_VALID_REQUEST:
		ma.handleValidationRequestMessage(msg)
	default:
		ma.handleRecordableMessage(msg)
	}
}

func (ma *MetaAgent) handleRecordableMessage(msg Message) {
	if ma.isSubsumed {
		ma.subsumedBy.handleRecordableMessage(msg)
		return
	}
	ma.messageStatistics.recordMessage(msg)
}

// Code for (1) Communication Partner Validation

func (ma *MetaAgent) setupCommunicationPartnerSearch() {
	ma.expectedComValidRequests = len(ma.subsumedModelAgents) * len(ma.externalModelAgents)
	ma.receivedComValidRequests = 0
	ma.isProcessingPartnerValidation = false
	ma.messageStatistics.clear()
}

func (ma *MetaAgent) handleCommunicationPartnerSearch() {
	ma.SignalMessagingComplete()
}

func (ma *MetaAgent) handleValidationRequestMessage(msg Message) {
	if ma.isSubsumed {
		if ma.serv.areInternalMessagesSynchronous {
			ma.SendSynchronousMessage(&msg, ma.subsumedBy.GetID())
		} else {
			ma.SendMessage(&msg, ma.subsumedBy.GetID())
		}
		return
	}
	ma.messageStatistics.recordMessage(msg)
	ma.receivedComValidRequests++
	messageDropsExpected := 0 // Temporary solution for handling dropped traffic
	if !ma.serv.areInternalMessagesSynchronous {
		messageDropsExpected = ma.expectedComValidRequests * (10 / ma.serv.GetAgentMessagingBandwidth())
	}
	if ma.receivedComValidRequests >= ma.expectedComValidRequests-messageDropsExpected && !ma.isProcessingPartnerValidation {
		ma.isProcessingPartnerValidation = true
		ma.messageStatistics.mutex.Lock()
		ma.processCommunicationPartnerValidation()
		ma.messageStatistics.mutex.Unlock()
	}
}

func (ma *MetaAgent) processCommunicationPartnerValidation() {
	responses, internal := ma.callPartnerSearch()
	for sender := range responses {
		senderAgent := ma.serv.modelAgentMap[sender]
		for receiver := range responses[sender] {
			if receiver == sender {
				continue
			}
			msg := senderAgent.createValidationMessage(responses[sender][receiver])
			if ma.serv.areInternalMessagesSynchronous {
				ma.serv.modelAgentMap[sender].SendSynchronousMessageSilently(msg, receiver)
			} else {
				ma.serv.modelAgentMap[sender].SendMessageSilently(msg, receiver)
			}
		}
	}
	for ag := range internal {
		subsumedAgent := ma.serv.modelAgentMap[ag]
		for _, partner := range internal[ag] {
			subsumedAgent.validComPartners = append(subsumedAgent.validComPartners, partner)
		}
	}
	ma.SignalMessagingComplete()
}

func (ma *MetaAgent) signalAllSubsumedMetaAgentsComplete() {
	for _, ag := range ma.subsumedMetaAgents {
		ag.SignalMessagingComplete()
	}
}

// Code for (2) Main Message Phase

func (ma *MetaAgent) setupMainCommunicationPhase() {
	ma.messageStatistics.clear()
	ma.subsumedAgentsFinishedMainPhase = 0
}

func (ma *MetaAgent) handleMainCommunicationPhase() {}

func (ma *MetaAgent) handleMainPhaseEndMessage() {
	ma.subsumedAgentsFinishedMainPhase++
	if ma.subsumedAgentsFinishedMainPhase >= len(ma.subsumedAgents) {
		if ma.isSubsumed {
			ma.subsumedBy.handleMainPhaseEndMessage()
		}
		ma.SignalMessagingComplete()
	}
}

// Code for (2) State Update Phase

func (ma *MetaAgent) setupStateUpdatePhase() {}

func (ma *MetaAgent) handleStateUpdatePhase() {
	if ma.isSubsumed {
		ma.SignalMessagingComplete()
		return
	}
	states := ma.callPredict()
	ma.state.applyStateChange(states)
	ma.forwardStatesToModelAgents(states)
	ma.VerifyAndDissolve()
	ma.informObserverAgents()
	ma.SignalMessagingComplete()
}

func (ma *MetaAgent) forwardStatesToModelAgents(states map[uuid.UUID][]byte) {
	for id, state := range states {
		msg := ma.CreateMessage()
		msg.MessageType = MSGTYPE_META_UPDATE_MODEL
		msg.Data = state
		msg.Recipient = id
		if ma.serv.areInternalMessagesSynchronous {
			ma.SendSynchronousMessage(msg, id)
		} else {
			ma.SendMessage(msg, id)
		}
	}
}

func (ma *MetaAgent) informObserverAgents() {
	msg := ma.CreateMessage()
	msg.MessageType = MSGTYPE_META_STATE_UPDATE
	for _, ag := range ma.serv.observerAgents {
		if ma.serv.areInternalMessagesSynchronous {
			ma.SendSynchronousMessage(msg, ag)
		} else {
			ma.SendMessage(msg, ag)
		}
	}
}

func (ma *MetaAgent) isSubsumedByMetaTree(ag uuid.UUID) bool {
	if slices.Contains(ma.subsumedAgents, ag) {
		return true
	}
	for _, child := range ma.subsumedMetaAgents {
		if child.isSubsumedByMetaTree(ag) {
			return true
		}
	}
	return false
}

func (ma *MetaAgent) getExternalModelAgents() []uuid.UUID {
	if ma.isSubsumed {
		return ma.subsumedBy.getExternalModelAgents()
	}
	return ma.externalModelAgents
}

// Exposed Functions

func (ma *MetaAgent) GetCondition() *MetaCondition {
	return &ma.condition
}

func (ma *MetaAgent) GetState() *MetaState {
	return &ma.state
}

func (ma *MetaAgent) GetMessageStatistics() *MessageStatistics {
	return &ma.messageStatistics
}

func (ma *MetaAgent) GetSubsumedAgents() *[]uuid.UUID {
	return &ma.subsumedAgents
}

func (ma *MetaAgent) GetAllSubsumedModelAgentsRecursive() []uuid.UUID {
	modelAgents := make([]uuid.UUID, 0)
	for ag := range ma.state.GetModelStatesRecursive() {
		modelAgents = append(modelAgents, ag)
	}
	return modelAgents
}

func (ma *MetaAgent) CreateMessage() *Message {
	return &Message{BaseMessage: ma.CreateBaseMessage(), MessageType: 0, Data: make([]byte, 0)}
}

func (ma *MetaAgent) Verify() bool {
	return ma.condition.verify(&ma.messageStatistics, &ma.state)
}

func (ma *MetaAgent) VerifyAndDissolve() {
	if !ma.condition.verify(&ma.messageStatistics, &ma.state) {
		ma.Dissolve()
	}
}

func (ma *MetaAgent) Dissolve() {
	for _, id := range ma.subsumedAgents {
		ma, ok := ma.serv.metaAgentMap[id]
		if ok {
			ma.VerifyAndDissolve()
		}
	}
	ma.hasDissolved = true
}

func (ma *MetaAgent) SetEvaluateFunc(evaluate func(*MetaAgent) float32) {
	ma.evaluate = evaluate
}

func (ma *MetaAgent) SetExplanationFunc(explanationFunc func(*MetaAgent)) {
	ma.condition.explain = explanationFunc
}

func (ma *MetaAgent) Evaluate() float32 {
	return ma.evaluate(ma)
}

func (ma *MetaAgent) Explain() {
	ma.condition.explain(ma)
}

// WIP

// handling nil partner search is wip
func (ma *MetaAgent) callPartnerSearch() (map[uuid.UUID]map[uuid.UUID]bool, map[uuid.UUID][]uuid.UUID) {
	if ma.partnerSearch == nil {
		responses := make(map[uuid.UUID]map[uuid.UUID]bool)
		internal := make(map[uuid.UUID][]uuid.UUID)
		for _, ag := range ma.subsumedMetaAgents {
			r, i := ag.callPartnerSearch()
			for subsumedAgent := range r {
				responses[subsumedAgent] = r[subsumedAgent]
				_, ok := internal[subsumedAgent]
				if !ok {
					internal[subsumedAgent] = i[subsumedAgent]
				} else {
					for _, partner := range i[subsumedAgent] {
						internal[subsumedAgent] = append(internal[subsumedAgent], partner)
					}
				}
				internal[subsumedAgent] = i[subsumedAgent]
			}
		}
		for _, ag := range ma.subsumedModelAgents {
			responses[ag.GetID()] = make(map[uuid.UUID]bool)
			_, ok := internal[ag.GetID()]
			if !ok {
				internal[ag.GetID()] = make([]uuid.UUID, 0)
			}
			receivedMsgs, _ := ma.messageStatistics.GetAllMessagesToAgent(ag.GetID())
			for _, msg := range receivedMsgs {
				responses[ag.GetID()][msg.GetSender()] = ag.validationFunc(msg)
			}
			for _, ag2 := range ma.subsumedModelAgents {
				if ag.GetID() == ag2.GetID() {
					continue
				}
				val := ag2.validationFunc(*ag.createValidationRequestMessage())
				if val {
					internal[ag.GetID()] = append(internal[ag.GetID()], ag2.GetID())
				}
			}
		}
		for _, metaAgent1 := range ma.subsumedMetaAgents {
			modelAgents1 := metaAgent1.state.GetModelStatesRecursive()
			for _, metaAgent2 := range ma.subsumedMetaAgents {
				if metaAgent1.GetID() == metaAgent2.GetID() {
					continue
				}
				modelAgents2 := metaAgent2.state.GetModelStatesRecursive()
				for ag1 := range modelAgents1 {
					_, ok := internal[ag1]
					if !ok {
						internal[ag1] = make([]uuid.UUID, 0)
					}
					for ag2 := range modelAgents2 {
						val := ma.serv.modelAgentMap[ag2].validationFunc(*ma.serv.modelAgentMap[ag1].createValidationRequestMessage())
						if val {
							internal[ag1] = append(internal[ag1], ag2)
						}
					}
				}
			}
		}
		return responses, internal
	}
	return ma.partnerSearch(&ma.messageStatistics, &ma.state)
}

// handling nil predict is wip
func (ma *MetaAgent) callPredict() map[uuid.UUID][]byte {
	if ma.predict == nil {
		predictions := make(map[uuid.UUID][]byte)
		for _, ag := range ma.subsumedMetaAgents {
			p := ag.callPredict()
			for subsumedAgent := range p {
				predictions[subsumedAgent] = p[subsumedAgent]
			}
		}
		for _, ag := range ma.subsumedModelAgents {
			predictions[ag.GetID()] = ag.stateUpdateFunc()
		}
	}
	return ma.predict(&ma.messageStatistics, &ma.state)
}
