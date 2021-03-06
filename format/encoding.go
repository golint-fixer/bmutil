// Copyright (c) 2015 Monetas.
// Copyright 2016 Daniel Krawisz.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package format

import (
	"errors"
	"fmt"
	"io"
	"regexp"

	"github.com/DanielKrawisz/bmutil"
	"github.com/DanielKrawisz/bmutil/format/serialize"
	"github.com/DanielKrawisz/bmutil/wire"
)

var encoding2Regex = regexp.MustCompile(`^Subject:(.*)\nBody:((?s).*)`)

// Encoding represents a msg or broadcast object payload.
type Encoding interface {
	Encoding() uint64
	encoding() serialize.Format
	Message() []byte
	readMessage([]byte) error
	ToProtobuf() *serialize.Encoding
}

// Encode encodes the Encoding to a writer.
func Encode(w io.Writer, l Encoding) error {
	var err error
	if err = bmutil.WriteVarInt(w, l.Encoding()); err != nil {
		return err
	}

	message := l.Message()
	msgLength := uint64(len(message))
	if err = bmutil.WriteVarInt(w, msgLength); err != nil {
		return err
	}
	if _, err := w.Write(message); err != nil {
		return err
	}

	return nil
}

// Encoding1 implements the Bitmessage interface and represents a
// MsgMsg or MsgBroadcast with encoding type 1.
type Encoding1 struct {
	Body string
}

// Encoding returns the encoding format of the bitmessage.
func (l *Encoding1) Encoding() uint64 {
	return 1
}

// Encoding returns the encoding format of the bitmessage.
func (l *Encoding1) encoding() serialize.Format {
	return serialize.Format_ENCODING1
}

// Message returns the raw form of the object payload.
func (l *Encoding1) Message() []byte {
	return []byte(l.Body)
}

// ReadMessage reads the object payload and incorporates it.
func (l *Encoding1) readMessage(msg []byte) error {
	l.Body = string(msg)
	return nil
}

// ToProtobuf encodes the message in a protobuf format.
func (l *Encoding1) ToProtobuf() *serialize.Encoding {
	return &serialize.Encoding{
		Format: l.encoding(),
		Body:   []byte(l.Body),
	}
}

// Encoding2 implements the Bitmessage interface and represents a
// MsgMsg or MsgBroadcast with encoding type 2. It also implements the
type Encoding2 struct {
	Subject string
	Body    string
}

// Encoding returns the encoding format of the bitmessage.
func (l *Encoding2) Encoding() uint64 {
	return 2
}

// Encoding returns the encoding format of the bitmessage.
func (l *Encoding2) encoding() serialize.Format {
	return serialize.Format_ENCODING2
}

// Message returns the raw form of the object payload.
func (l *Encoding2) Message() []byte {
	return []byte(fmt.Sprintf("Subject:%s\nBody:%s", l.Subject, l.Body))
}

// ReadMessage reads the object payload and incorporates it.
func (l *Encoding2) readMessage(msg []byte) error {
	matches := encoding2Regex.FindStringSubmatch(string(msg))
	if len(matches) < 3 {
		return errors.New("Invalid format")
	}
	l.Subject = matches[1]
	l.Body = matches[2]
	return nil
}

// ToProtobuf encodes the message in a protobuf format.
func (l *Encoding2) ToProtobuf() *serialize.Encoding {
	return &serialize.Encoding{
		Format:  l.encoding(),
		Subject: []byte(l.Subject),
		Body:    []byte(l.Body),
	}
}

// Read takes an encoding format code and an object payload and
// returns it as an Encoding object.
func Read(encoding uint64, msg []byte) (Encoding, error) {
	var q Encoding
	switch encoding {
	case 1:
		q = &Encoding1{}
	case 2:
		q = &Encoding2{}
	default:
		return nil, errors.New("Unsupported encoding")
	}
	err := q.readMessage(msg)
	if err != nil {
		return nil, err
	}
	return q, nil
}

// Decode reads an Encoding type from a stream.
func Decode(r io.Reader) (Encoding, error) {
	var encoding uint64
	var err error
	if encoding, err = bmutil.ReadVarInt(r); err != nil {
		return nil, err
	}
	var msgLength uint64
	if msgLength, err = bmutil.ReadVarInt(r); err != nil {
		return nil, err
	}
	if msgLength > wire.MaxPayloadOfMsgObject {
		str := fmt.Sprintf("message length exceeds max length - "+
			"indicates %d, but max length is %d",
			msgLength, wire.MaxPayloadOfMsgObject)
		return nil, wire.NewMessageError("DecodeFromDecrypted", str)
	}
	message := make([]byte, msgLength)
	_, err = io.ReadFull(r, message)
	if err != nil {
		return nil, err
	}

	return Read(encoding, message)
}
