package SOMACS

import "github.com/google/uuid"

type MetaState struct {
	ChildStates map[uuid.UUID]*MetaState
	ModelStates map[uuid.UUID][]byte
}

func (ms *MetaState) createMetaState(modelAgentStates map[uuid.UUID][]byte, metaAgentStates map[uuid.UUID]*MetaState) {
	ms.ChildStates = metaAgentStates
	ms.ModelStates = modelAgentStates
}

func (ms *MetaState) applyStateChange(states map[uuid.UUID][]byte) {
	for id := range ms.ModelStates {
		state, ok := states[id]
		if !ok {
			continue
		}
		ms.ModelStates[id] = state
	}
	for _, st := range ms.ChildStates {
		st.applyStateChange(states)
	}
}

// Exposed Functions

func (ms *MetaState) GetModelStatesRecursive() map[uuid.UUID][]byte {
	flatStates := make(map[uuid.UUID][]byte)
	for id, st := range ms.ModelStates {
		flatStates[id] = st
	}
	for _, st := range ms.ChildStates {
		states := st.GetModelStatesRecursive()
		for id, st := range states {
			flatStates[id] = st
		}
	}
	return flatStates
}
