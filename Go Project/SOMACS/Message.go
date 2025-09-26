package SOMACS

import (
	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/message"
	"github.com/google/uuid"
)

const MSGTYPE_DEFAULT = 0
const MSGTYPE_COM_VALID_REQUEST = -1
const MSGTYPE_COM_VALID = -2
const MSGTYPE_COM_MAIN_END = -3
const MSGTYPE_COM_STATE_UPDATE = -4
const MSGTYPE_META_UPDATE_MODEL = -5
const MSGTYPE_META_STATE_UPDATE = -6

type Message struct {
	message.BaseMessage
	MessageType int
	Data        []byte
	Recipient   uuid.UUID
}

func (d Message) InvokeMessageHandler(recipient IGenericAgent) {
	if recipient == nil {
		return
	}
	recipient.handleMessage(d)
}
