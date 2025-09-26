package SOMACS

import (
	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/agent"
	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/message"
	"github.com/google/uuid"
)

type ModelAgent struct {
	*agent.BaseAgent[IGenericAgent]
	modelAgents                    *[]uuid.UUID
	observerAgents                 *[]uuid.UUID
	environment                    *map[string][]byte
	areInternalMessagesSynchronous *bool

	state []byte

	// For (1) Communication Partner Search
	expectedComValidRequests  int
	receivedComValidRequests  int
	expectedComValidResponses int
	receivedComValidResponses int
	validComPartners          []uuid.UUID
	validationFunc            func(Message) bool
	validationRequestData     []byte

	// For (2) Main Communication Phase
	finishedMainComPhase bool

	// For (3) State Update
	stateUpdateFunc func() []byte

	// For meta agent subsumption
	isSubsumed bool
	subsumedBy *MetaAgent

	// Package Exposure
	OnHandleMessage                   Event[Message]
	OnSetupCommunicationPartnerSearch Event[*ModelAgent]
	OnSetupMainCommunicationPhase     Event[*ModelAgent]
	OnSetupStateUpdatePhase           Event[*ModelAgent]

	OnBeginMainCommunicationPhase Event[*ModelAgent]
}

func CreateModelAgentBase(serv *Server) *ModelAgent {
	ma := &ModelAgent{BaseAgent: agent.CreateBaseAgent(serv)}
	ma.modelAgents = &serv.modelAgents
	ma.observerAgents = &serv.observerAgents
	ma.environment = &serv.environmentVariables

	ma.state = make([]byte, 0)

	ma.areInternalMessagesSynchronous = &serv.areInternalMessagesSynchronous
	ma.validComPartners = make([]uuid.UUID, 0, len(*ma.modelAgents))
	ma.validationFunc = func(Message) bool { return true }
	ma.validationRequestData = make([]byte, 0)

	ma.finishedMainComPhase = false

	ma.stateUpdateFunc = func() []byte { return make([]byte, 0) }

	ma.isSubsumed = false
	ma.subsumedBy = nil

	serv.modelAgents = append(serv.modelAgents, ma.GetID())
	serv.modelAgentMap[ma.GetID()] = ma
	return ma
}

func createModelAgent(serv *Server) IGenericAgent {
	return CreateModelAgentBase(serv)
}

func (ma *ModelAgent) handleMessage(msg Message) {
	switch msg.MessageType {
	case MSGTYPE_COM_VALID_REQUEST:
		ma.handleValidationRequestMessage(msg)
		return
	case MSGTYPE_COM_VALID:
		ma.handleValidationMessage(msg)
		return
	case MSGTYPE_META_UPDATE_MODEL:
		ma.handleMetaUpdateMessage(msg)
		return
	default:
		if ma.isSubsumed {
			if *ma.areInternalMessagesSynchronous {
				ma.SendSynchronousMessageSilently(&msg, ma.subsumedBy.GetID())
			} else {
				ma.SendMessageSilently(&msg, ma.subsumedBy.GetID())
			}
		}
	}
	ma.OnHandleMessage.invoke(msg)
}

// Code for (1) Communication Partner Validation

func (ma *ModelAgent) setupCommunicationPartnerSearch() {
	ma.OnSetupCommunicationPartnerSearch.invoke(ma)
	if ma.isSubsumed {
		ma.expectedComValidRequests = len(ma.subsumedBy.getExternalModelAgents())
		ma.expectedComValidResponses = len(ma.subsumedBy.getExternalModelAgents())
	} else {
		ma.expectedComValidRequests = len(*ma.modelAgents) - 1
		ma.expectedComValidResponses = len(*ma.modelAgents) - 1
	}
	ma.receivedComValidRequests = 0
	ma.receivedComValidResponses = 0
	ma.validComPartners = ma.validComPartners[:0]
}

func (ma *ModelAgent) handleCommunicationPartnerSearch() {
	msg := ma.createValidationRequestMessage()
	msg.Data = ma.validationRequestData
	if ma.isSubsumed {
		if *ma.areInternalMessagesSynchronous {
			ma.BroadcastSynchronousMessageSilentlyToRecipients(msg, ma.subsumedBy.getExternalModelAgents())
			return
		}
		ma.BroadcastMessageSilentlyToRecipients(msg, ma.subsumedBy.getExternalModelAgents())
		return
	}
	if *ma.areInternalMessagesSynchronous {
		ma.BroadcastSynchronousMessageSilently(msg)
		return
	}
	ma.BroadcastMessageSilently(msg)
}

func (ma *ModelAgent) createValidationRequestMessage() *Message {
	msg := ma.CreateMessage()
	msg.MessageType = MSGTYPE_COM_VALID_REQUEST
	return msg
}

func (ma *ModelAgent) handleValidationRequestMessage(msg Message) {
	ma.receivedComValidRequests++
	if ma.isSubsumed {
		if *ma.areInternalMessagesSynchronous {
			ma.SendSynchronousMessageSilently(&msg, ma.subsumedBy.GetID())
		} else {
			ma.SendMessageSilently(&msg, ma.subsumedBy.GetID())
		}
		return
	}
	isValid := ma.validationFunc(msg)
	response := ma.createValidationMessage(isValid)
	if *ma.areInternalMessagesSynchronous {
		ma.SendSynchronousMessageSilently(response, msg.GetSender())
	} else {
		ma.SendMessageSilently(response, msg.GetSender())
	}
	ma.checkCommunicationPartnerSearchEnd()
}

func (ma *ModelAgent) createValidationMessage(isValid bool) *Message {
	msg := ma.CreateMessage()
	msg.MessageType = MSGTYPE_COM_VALID
	if isValid {
		msg.Data = append(msg.Data, 1)
	} else {
		msg.Data = append(msg.Data, 0)
	}
	return msg
}

func (ma *ModelAgent) handleValidationMessage(msg Message) {
	if len(msg.Data) == 0 {
		panic("Communication Partner Validation Message with no Data found - did you compose the Message?")
	}
	isValid := msg.Data[0] != 0
	if isValid {
		ma.validComPartners = append(ma.validComPartners, msg.GetSender())
	}
	ma.receivedComValidResponses++
	ma.checkCommunicationPartnerSearchEnd()
}

func (ma *ModelAgent) checkCommunicationPartnerSearchEnd() {
	if ma.receivedComValidRequests >= ma.expectedComValidRequests && ma.receivedComValidResponses >= ma.expectedComValidResponses {
		ma.SignalMessagingComplete()
	}
}

// Code for (2) Main Message Phase

func (ma *ModelAgent) setupMainCommunicationPhase() {
	ma.OnSetupMainCommunicationPhase.invoke(ma)
	ma.finishedMainComPhase = false
}

func (ma *ModelAgent) handleMainCommunicationPhase() {
	ok := ma.OnBeginMainCommunicationPhase.invoke(ma)
	if !ok {
		ma.SignalMessagingComplete()
	}
}

// Code for (3) State Update Phase

func (ma *ModelAgent) setupStateUpdatePhase() {
	ma.OnSetupStateUpdatePhase.invoke(ma)
}

func (ma *ModelAgent) handleStateUpdatePhase() {
	if ma.isSubsumed {
		return
	}
	ma.state = ma.stateUpdateFunc()
	msg := ma.createStateUpdateMessage()
	if *ma.areInternalMessagesSynchronous {
		ma.SendSynchronousMessageToObservers(msg)
	} else {
		ma.SendMessageToObservers(msg)
	}
	ma.SignalMessagingComplete()
}

func (ma *ModelAgent) createStateUpdateMessage() *Message {
	msg := ma.CreateMessage()
	msg.MessageType = MSGTYPE_COM_STATE_UPDATE
	msg.Data = ma.state
	return msg
}

func (ma *ModelAgent) handleMetaUpdateMessage(msg Message) {
	ma.state = msg.Data
	ma.SignalMessagingComplete()
}

// Model Agent Messaging Overwrites

func (ma *ModelAgent) SendMessage(msg message.IMessage[IGenericAgent], recipient uuid.UUID) {
	ma.BaseAgent.SendMessage(msg, recipient)
	for _, oa := range *ma.observerAgents {
		ma.BaseAgent.SendMessage(msg, oa)
	}
}

func (ma *ModelAgent) SendMessageToObservers(msg message.IMessage[IGenericAgent]) {
	for _, oa := range *ma.observerAgents {
		ma.BaseAgent.SendMessage(msg, oa)
	}
}

func (ma *ModelAgent) SendMessageSilently(msg message.IMessage[IGenericAgent], recipient uuid.UUID) {
	ma.BaseAgent.SendMessage(msg, recipient)
}

func (ma *ModelAgent) BroadcastMessageToRecipients(msg message.IMessage[IGenericAgent], recipients []uuid.UUID) {
	typedMsg, ok := msg.(*Message)
	for _, recipient := range recipients {
		if !ok {
			ma.SendMessage(msg, recipient)
			continue
		}
		typedMsg.Recipient = recipient
		ma.SendMessage(typedMsg, recipient)
	}
}

func (ma *ModelAgent) BroadcastMessageSilently(msg message.IMessage[IGenericAgent]) {
	if msg.GetSender() == uuid.Nil {
		panic("No sender found - did you compose the BaseMessage?")
	}
	for _, recipient := range *ma.modelAgents {
		if recipient == msg.GetSender() {
			continue
		}
		ma.BaseAgent.SendMessage(msg, recipient)
	}
}

func (ma *ModelAgent) BroadcastMessageSilentlyToRecipients(msg message.IMessage[IGenericAgent], recipients []uuid.UUID) {
	typedMsg, ok := msg.(*Message)
	for _, recipient := range recipients {
		if !ok {
			ma.BaseAgent.SendMessage(msg, recipient)
			continue
		}
		typedMsg.Recipient = recipient
		ma.BaseAgent.SendMessage(typedMsg, recipient)
	}
}

func (ma *ModelAgent) SendSynchronousMessage(msg message.IMessage[IGenericAgent], recipient uuid.UUID) {
	ma.BaseAgent.SendSynchronousMessage(msg, recipient)
	for _, oa := range *ma.observerAgents {
		ma.BaseAgent.SendSynchronousMessage(msg, oa)
	}
}

func (ma *ModelAgent) SendSynchronousMessageToObservers(msg message.IMessage[IGenericAgent]) {
	for _, oa := range *ma.observerAgents {
		ma.BaseAgent.SendSynchronousMessage(msg, oa)
	}
}

func (ma *ModelAgent) SendSynchronousMessageSilently(msg message.IMessage[IGenericAgent], recipient uuid.UUID) {
	ma.BaseAgent.SendSynchronousMessage(msg, recipient)
}

func (ma *ModelAgent) BroadcastSynchronousMessageToRecipients(msg message.IMessage[IGenericAgent], recipients []uuid.UUID) {
	typedMsg, ok := msg.(*Message)
	for _, recipient := range recipients {
		if !ok {
			ma.SendSynchronousMessage(msg, recipient)
			continue
		}
		typedMsg.Recipient = recipient
		ma.SendSynchronousMessage(typedMsg, recipient)
	}
}

func (ma *ModelAgent) BroadcastSynchronousMessageSilently(msg message.IMessage[IGenericAgent]) {
	if msg.GetSender() == uuid.Nil {
		panic("No sender found - did you compose the BaseMessage?")
	}
	for _, recipient := range *ma.modelAgents {
		if recipient == msg.GetSender() {
			continue
		}
		ma.BaseAgent.SendSynchronousMessage(msg, recipient)
	}
}

func (ma *ModelAgent) BroadcastSynchronousMessageSilentlyToRecipients(msg message.IMessage[IGenericAgent], recipients []uuid.UUID) {
	typedMsg, ok := msg.(*Message)
	for _, recipient := range recipients {
		if !ok {
			ma.BaseAgent.SendSynchronousMessage(msg, recipient)
			continue
		}
		typedMsg.Recipient = recipient
		ma.BaseAgent.SendSynchronousMessage(typedMsg, recipient)
	}
}

// Exposed Functions

func (ma *ModelAgent) CreateMessage() *Message {
	return &Message{BaseMessage: ma.CreateBaseMessage(), MessageType: 0, Data: make([]byte, 0)}
}

func (ma *ModelAgent) EndMainCommunicationPhase() {
	ma.finishedMainComPhase = true
	msg := ma.CreateMessage()
	msg.MessageType = MSGTYPE_COM_MAIN_END
	ma.SendMessageToObservers(msg)
	if ma.isSubsumed {
		ma.SendMessageSilently(msg, ma.subsumedBy.GetID())
	}
	ma.SignalMessagingComplete()
}

func (ma *ModelAgent) GetEnvironmentVariable(key string) ([]byte, bool) {
	ret, ok := (*ma.environment)[key]
	return ret, ok
}

func (ma *ModelAgent) GetState() []byte {
	return ma.state
}

func (ma *ModelAgent) GetValidCommunicationPartners() []uuid.UUID {
	return ma.validComPartners
}

func (ma *ModelAgent) GetValidationRequestData() []byte {
	return ma.validationRequestData
}

func (ma *ModelAgent) ValidateMessage(msg Message) bool {
	return ma.validationFunc(msg)
}

func (ma *ModelAgent) SetValidationFunc(validationFunc func(Message) bool) {
	ma.validationFunc = validationFunc
}

func (ma *ModelAgent) SetValidationRequestData(validationRequestData []byte) {
	ma.validationRequestData = validationRequestData
}

func (ma *ModelAgent) SetStateUpdateFunc(stateUpdateFunc func() []byte) {
	ma.stateUpdateFunc = stateUpdateFunc
}
