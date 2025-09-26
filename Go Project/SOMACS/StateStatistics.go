package SOMACS

import (
	"github.com/google/uuid"
	"sync"
)

type StateStatistics struct {
	states        map[uuid.UUID][]byte
	metaStates    map[uuid.UUID]*MetaState
	metaDissolved map[uuid.UUID]bool
	mutex         sync.Mutex
}

func (ss *StateStatistics) createStateStatistics() {
	ss.states = make(map[uuid.UUID][]byte)
	ss.metaStates = make(map[uuid.UUID]*MetaState)
	ss.metaDissolved = make(map[uuid.UUID]bool)
}

func (ss *StateStatistics) recordState(msg Message) {
	ss.mutex.Lock()
	ss.states[msg.GetSender()] = msg.Data
	ss.mutex.Unlock()
}

func (ss *StateStatistics) recordMeta(id uuid.UUID, state *MetaState, dissolved bool) {
	ss.mutex.Lock()
	ss.metaStates[id] = state
	ss.metaDissolved[id] = dissolved
	ss.mutex.Unlock()
}

func (ss *StateStatistics) clear() {
	ss.mutex.Lock()
	for id := range ss.states {
		delete(ss.states, id)
	}
	for id := range ss.metaStates {
		delete(ss.metaStates, id)
	}
	for id := range ss.metaDissolved {
		delete(ss.metaDissolved, id)
	}
	ss.mutex.Unlock()
}

// Exposed Functions

func (ss *StateStatistics) ClearEmptyStates() {
	ss.mutex.Lock()
	for id := range ss.states {
		if len(ss.states[id]) == 0 {
			delete(ss.states, id)
		}
	}
	ss.mutex.Unlock()
}

func (ss *StateStatistics) GetStates() map[uuid.UUID][]byte {
	return ss.states
}

func (ss *StateStatistics) GetMetaStates() map[uuid.UUID]*MetaState {
	return ss.metaStates
}

func (ss *StateStatistics) GetMetaDissolved() map[uuid.UUID]bool {
	return ss.metaDissolved
}
