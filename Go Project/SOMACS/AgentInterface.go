package SOMACS

import (
	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/agent"
)

type IGenericAgent interface {
	agent.IAgent[IGenericAgent]
	CreateMessage() *Message
	handleMessage(Message)
	setupCommunicationPartnerSearch()
	handleCommunicationPartnerSearch()
	setupMainCommunicationPhase()
	handleMainCommunicationPhase()
	setupStateUpdatePhase()
	handleStateUpdatePhase()
}
