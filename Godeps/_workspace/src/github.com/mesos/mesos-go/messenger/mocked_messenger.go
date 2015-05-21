/**
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package messenger

import (
	"reflect"

	"github.com/gogo/protobuf/proto"
	"github.com/mesos/mesos-go/upid"
	"github.com/stretchr/testify/mock"
	"golang.org/x/net/context"
)

type message struct {
	from *upid.UPID
	msg  proto.Message
}

// MockedMessenger is a messenger that returns error on every operation.
type MockedMessenger struct {
	mock.Mock
	messageQueue chan *message
	handlers     map[string]MessageHandler
	stop         chan struct{}
}

// NewMockedMessenger returns a mocked messenger used for testing.
func NewMockedMessenger() *MockedMessenger {
	return &MockedMessenger{
		messageQueue: make(chan *message, 1),
		handlers:     make(map[string]MessageHandler),
		stop:         make(chan struct{}),
	}
}

// Install is a mocked implementation.
func (m *MockedMessenger) Install(handler MessageHandler, msg proto.Message) error {
	m.handlers[reflect.TypeOf(msg).Elem().Name()] = handler
	return m.Called().Error(0)
}

// Send is a mocked implementation.
func (m *MockedMessenger) Send(ctx context.Context, upid *upid.UPID, msg proto.Message) error {
	return m.Called().Error(0)
}

func (m *MockedMessenger) Route(ctx context.Context, upid *upid.UPID, msg proto.Message) error {
	return m.Called().Error(0)
}

// Start is a mocked implementation.
func (m *MockedMessenger) Start() error {
	go m.recvLoop()
	return m.Called().Error(0)
}

// Stop is a mocked implementation.
func (m *MockedMessenger) Stop() error {
	// don't close an already-closed channel
	select {
	case <-m.stop:
		// noop
	default:
		close(m.stop)
	}
	return m.Called().Error(0)
}

// UPID is a mocked implementation.
func (m *MockedMessenger) UPID() *upid.UPID {
	return m.Called().Get(0).(*upid.UPID)
}

func (m *MockedMessenger) recvLoop() {
	for {
		select {
		case <-m.stop:
			return
		case msg := <-m.messageQueue:
			name := reflect.TypeOf(msg.msg).Elem().Name()
			m.handlers[name](msg.from, msg.msg)
		}
	}
}

// Recv receives a upid and a message, it will dispatch the message to its handler
// with the upid. This is for testing.
func (m *MockedMessenger) Recv(from *upid.UPID, msg proto.Message) {
	m.messageQueue <- &message{from, msg}
}
