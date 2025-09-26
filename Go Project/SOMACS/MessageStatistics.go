package SOMACS

import (
	"github.com/google/uuid"
	"sync"
)

type MessageStatistics struct {
	communicationMap                 map[uuid.UUID]map[uuid.UUID][]Message //map[from][to][]messages
	hasSignaledMainMessagingComplete map[uuid.UUID]bool
	mutex                            sync.Mutex
}

func (ms *MessageStatistics) createMessageStatistics() {
	ms.communicationMap = make(map[uuid.UUID]map[uuid.UUID][]Message)
	ms.hasSignaledMainMessagingComplete = make(map[uuid.UUID]bool)
}

func (ms *MessageStatistics) recordMessage(msg Message) {
	ms.mutex.Lock()
	comMap, ok := ms.communicationMap[msg.GetSender()]
	if ok {
		_, ok2 := comMap[msg.Recipient]
		if !ok2 {
			comMap[msg.Recipient] = make([]Message, 0, 16)
		}
		comMap[msg.Recipient] = append(comMap[msg.Recipient], msg)
	} else {
		comMap = make(map[uuid.UUID][]Message)
		comMap[msg.Recipient] = make([]Message, 0, 16)
		comMap[msg.Recipient] = append(comMap[msg.Recipient], msg)
	}
	ms.communicationMap[msg.GetSender()] = comMap
	ms.mutex.Unlock()
}

func (ms *MessageStatistics) recordSignaledMainMessagingComplete(agent uuid.UUID) {
	ms.mutex.Lock()
	ms.hasSignaledMainMessagingComplete[agent] = true
	ms.mutex.Unlock()
}

func (ms *MessageStatistics) clear() {
	ms.mutex.Lock()
	for id := range ms.communicationMap {
		delete(ms.communicationMap, id)
	}
	for id := range ms.hasSignaledMainMessagingComplete {
		delete(ms.hasSignaledMainMessagingComplete, id)
	}
	ms.mutex.Unlock()
}

// Exposed Functions

func (ms *MessageStatistics) GetCommunicationMap() map[uuid.UUID]map[uuid.UUID][]Message {
	return ms.communicationMap
}

func (ms *MessageStatistics) HasCommunicated(sender uuid.UUID) bool {
	comMap, ok := ms.communicationMap[sender]
	if !ok {
		return false
	}
	return len(comMap[sender]) > 0
}

func (ms *MessageStatistics) GetAllMessagesFromAgent(sender uuid.UUID) ([]Message, bool) {
	comMap, ok := ms.communicationMap[sender]
	if !ok {
		return nil, false
	}
	messages := make([]Message, 0, 64)
	for recipient := range comMap {
		for _, msg := range comMap[recipient] {
			messages = append(messages, msg)
		}
	}
	return messages, true
}

func (ms *MessageStatistics) GetAllMessagesToAgent(recipient uuid.UUID) ([]Message, bool) {
	messages := make([]Message, 0, 64)
	for sender := range ms.communicationMap {
		for _, msg := range ms.communicationMap[sender][recipient] {
			messages = append(messages, msg)
		}
	}
	return messages, len(messages) > 0
}

func (ms *MessageStatistics) GetMessagesFromTo(sender, recipient uuid.UUID) ([]Message, bool) {
	comMap, ok := ms.communicationMap[sender]
	if !ok {
		return nil, false
	}
	comSlice, ok2 := comMap[recipient]
	if !ok2 {
		return nil, false
	}
	return comSlice, true
}

func (ms *MessageStatistics) GetMessagesOfTypeFromAgent(sender uuid.UUID, msgType int) ([]Message, bool) {
	comMap, ok := ms.communicationMap[sender]
	if !ok {
		return nil, false
	}
	messages := make([]Message, 0, 64)
	for recipient := range comMap {
		for _, msg := range comMap[recipient] {
			if msg.MessageType != msgType {
				continue
			}
			messages = append(messages, msg)
		}
	}
	return messages, true
}

func (ms *MessageStatistics) GetMessagesOfTypeToAgent(recipient uuid.UUID, msgType int) ([]Message, bool) {
	messages := make([]Message, 0, 64)
	for sender := range ms.communicationMap {
		for _, msg := range ms.communicationMap[sender][recipient] {
			if msg.MessageType != msgType {
				continue
			}
			messages = append(messages, msg)
		}
	}
	return messages, len(messages) > 0
}

func (ms *MessageStatistics) HasSentMessageTo(sender, recipient uuid.UUID) bool {
	comMap, ok := ms.communicationMap[sender]
	if !ok {
		return false
	}
	comSlice, ok2 := comMap[recipient]
	if !ok2 {
		return false
	}
	return len(comSlice) > 0
}

func (ms *MessageStatistics) HasReceivedMessageFrom(recipient, sender uuid.UUID) bool {
	return ms.HasSentMessageTo(sender, recipient)
}

func (ms *MessageStatistics) HasCommunicatedWith(agent1, agent2 uuid.UUID) bool {
	return ms.HasSentMessageTo(agent1, agent2) || ms.HasReceivedMessageFrom(agent1, agent2)
}

func (ms *MessageStatistics) HasCommunicatedWithAny(sender uuid.UUID, recipients []uuid.UUID) bool {
	for _, recipient := range recipients {
		if ms.HasCommunicatedWith(sender, recipient) {
			return true
		}
	}
	return false
}

func (ms *MessageStatistics) HasCommunicatedWithAll(sender uuid.UUID, recipients []uuid.UUID) bool {
	for _, recipient := range recipients {
		if !ms.HasCommunicatedWith(sender, recipient) {
			return false
		}
	}
	return true
}

func (ms *MessageStatistics) HasSignaledMainMessagingComplete(agent uuid.UUID) bool {
	val, ok := ms.hasSignaledMainMessagingComplete[agent]
	if !ok {
		return false
	}
	return val
}
