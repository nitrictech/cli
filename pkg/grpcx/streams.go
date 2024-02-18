// Copyright Nitric Pty Ltd.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package grpcx

import (
	"github.com/samber/lo"
	"google.golang.org/grpc"

	"github.com/nitrictech/nitric/core/pkg/workers"
)

type GrpcBidiStreamServer[ServerMessage any, ClientMessage any] interface {
	Send(ServerMessage) error
	Recv() (ClientMessage, error)
	grpc.ServerStream
}

type PeekableStreamServer[ServerMessage any, ClientMessage any] struct {
	buffer []lo.Tuple2[ClientMessage, error]
	GrpcBidiStreamServer[ServerMessage, ClientMessage]
}

func NewPeekableStreamServer[ServerMessage any, ClientMessage any](stream GrpcBidiStreamServer[ServerMessage, ClientMessage]) *PeekableStreamServer[ServerMessage, ClientMessage] {
	return &PeekableStreamServer[ServerMessage, ClientMessage]{
		buffer:               make([]lo.Tuple2[ClientMessage, error], 0),
		GrpcBidiStreamServer: stream,
	}
}

var _ workers.GrpcBidiStreamServer[workers.IdentifiableMessage, workers.IdentifiableMessage] = (*PeekableStreamServer[workers.IdentifiableMessage, workers.IdentifiableMessage])(nil)

func (s *PeekableStreamServer[ServerMessage, ClientMessage]) Recv() (ClientMessage, error) {
	if len(s.buffer) > 0 {
		msgToSend := s.buffer[0]
		s.buffer = s.buffer[1:]

		return msgToSend.A, msgToSend.B
	}

	return s.GrpcBidiStreamServer.Recv()
}

func (s *PeekableStreamServer[ServerMessage, ClientMessage]) Peek() (ClientMessage, error) {
	popped, err := s.GrpcBidiStreamServer.Recv()
	s.buffer = append(s.buffer, lo.T2(popped, err))
	return popped, err
}
