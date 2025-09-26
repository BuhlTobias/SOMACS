package SOMACS

import (
	"fmt"
	"github.com/google/uuid"
)

type MetaCondition struct {
	verifyFunc func(*MessageStatistics, *MetaState, map[uuid.UUID][]byte) bool
	baseState  map[uuid.UUID][]byte
	explain    func(*MetaAgent)
}

func (mc *MetaCondition) createMetaCondition(verify func(*MessageStatistics, *MetaState, map[uuid.UUID][]byte) bool, baseState map[uuid.UUID][]byte) {
	mc.verifyFunc = verify
	mc.baseState = baseState
	mc.explain = func(*MetaAgent) {
		fmt.Printf("No Explanation for MetaCondition provided.\n")
	}
}

func (mc *MetaCondition) verify(messageStatistics *MessageStatistics, currentState *MetaState) bool {
	if mc.verifyFunc == nil {
		return true
	}
	return mc.verifyFunc(messageStatistics, currentState, mc.baseState)
}

// Exposed functions

func (mc *MetaCondition) GetBaseState() map[uuid.UUID][]byte {
	return mc.baseState
}
