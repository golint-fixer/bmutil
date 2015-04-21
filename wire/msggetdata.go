package wire

import (
	"fmt"
	"io"

	"github.com/monetas/bmutil"
)

// MsgGetData implements the Message interface and represents a bitmessage
// getdata message. It is used to request data such as messages and broadcasts
// from another peer. It should be used in response to the inv (MsgInv) message
// to request the actual data referenced by each inventory vector the receiving
// peer doesn't already have. Each message is limited to a maximum number of
// inventory vectors, which is currently 50,000. As a result, multiple messages
// must be used to request larger amounts of data.
//
// Use the AddInvVect function to build up the list of inventory vectors when
// sending a getdata message to another peer.
type MsgGetData struct {
	InvList []*InvVect
}

// AddInvVect adds an inventory vector to the message.
func (msg *MsgGetData) AddInvVect(iv *InvVect) error {
	if len(msg.InvList)+1 > MaxInvPerMsg {
		str := fmt.Sprintf("too many invvect in message [max %v]",
			MaxInvPerMsg)
		return messageError("MsgGetData.AddInvVect", str)
	}

	msg.InvList = append(msg.InvList, iv)
	return nil
}

// Decode decodes r using the bitmessage protocol encoding into the receiver.
// This is part of the Message interface implementation.
func (msg *MsgGetData) Decode(r io.Reader) error {
	count, err := bmutil.ReadVarInt(r)
	if err != nil {
		return err
	}

	// Limit to max inventory vectors per message.
	if count > MaxInvPerMsg {
		str := fmt.Sprintf("too many invvect in message [%v]", count)
		return messageError("MsgGetData.Decode", str)
	}

	msg.InvList = make([]*InvVect, 0, count)
	for i := uint64(0); i < count; i++ {
		iv := InvVect{}
		err := readInvVect(r, &iv)
		if err != nil {
			return err
		}
		msg.AddInvVect(&iv)
	}

	return nil
}

// Encode encodes the receiver to w using the bitmessage protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgGetData) Encode(w io.Writer) error {
	// Limit to max inventory vectors per message.
	count := len(msg.InvList)
	if count > MaxInvPerMsg {
		str := fmt.Sprintf("too many invvect in message [%v]", count)
		return messageError("MsgGetData.Encode", str)
	}

	err := bmutil.WriteVarInt(w, uint64(count))
	if err != nil {
		return err
	}

	for _, iv := range msg.InvList {
		err := writeInvVect(w, iv)
		if err != nil {
			return err
		}
	}

	return nil
}

// Command returns the protocol command string for the message. This is part
// of the Message interface implementation.
func (msg *MsgGetData) Command() string {
	return CmdGetData
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver. This is part of the Message interface implementation.
func (msg *MsgGetData) MaxPayloadLength() int {
	// Num inventory vectors (varInt) + max allowed inventory vectors.
	return bmutil.MaxVarIntSize + (MaxInvPerMsg * maxInvVectPayload)
}

// NewMsgGetData returns a new bitmessage getdata message that conforms to the
// Message interface. See MsgGetData for details.
func NewMsgGetData() *MsgGetData {
	return &MsgGetData{
		InvList: make([]*InvVect, 0, defaultInvListAlloc),
	}
}

// NewMsgGetDataSizeHint returns a new bitmessage getdata message that conforms to
// the Message interface. See MsgGetData for details. This function differs
// from NewMsgGetData in that it allows a default allocation size for the
// backing array which houses the inventory vector list. This allows callers
// who know in advance how large the inventory list will grow to avoid the
// overhead of growing the internal backing array several times when appending
// large amounts of inventory vectors with AddInvVect. Note that the specified
// hint is just that - a hint that is used for the default allocation size.
// Adding more (or less) inventory vectors will still work properly. The size
// hint is limited to MaxInvPerMsg.
func NewMsgGetDataSizeHint(sizeHint uint) *MsgGetData {
	// Limit the specified hint to the maximum allow per message.
	if sizeHint > MaxInvPerMsg {
		sizeHint = MaxInvPerMsg
	}

	return &MsgGetData{
		InvList: make([]*InvVect, 0, sizeHint),
	}
}
