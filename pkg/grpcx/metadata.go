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
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const ServiceNameKey = "x-nitric-service-name"

type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context {
	return w.ctx
}

// newWrappedStream creates a new wrappedStream instance
func newWrappedStream(stream grpc.ServerStream, ctx context.Context) grpc.ServerStream {
	return &wrappedStream{ServerStream: stream, ctx: ctx}
}

func CreateServiceNameInterceptor(serviceName string) (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
			// Inject the name of the service
			md, _ := metadata.FromIncomingContext(ctx)
			md.Append(ServiceNameKey, serviceName) // example of adding new metadata

			newCtx := metadata.NewIncomingContext(ctx, md)

			return handler(newCtx, req)
		}, func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
			md, ok := metadata.FromIncomingContext(ss.Context())
			if !ok {
				md = metadata.MD{}
			}

			// Modify metadata here
			md.Append(ServiceNameKey, serviceName)

			// Create a new context with the modified metadata
			newCtx := metadata.NewIncomingContext(ss.Context(), md)

			// Create a new wrapped stream with the new context
			wrappedStream := newWrappedStream(ss, newCtx)

			// Call the original handler with the new wrapped stream
			return handler(srv, wrappedStream)
		}
}

// GetServiceNameFromIncomingContext extracts the nitric service name from the incoming context of a grpc request
func GetServiceNameFromIncomingContext(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", fmt.Errorf("request ctx missing expected metadata")
	}

	serviceName := strings.Join(md.Get(ServiceNameKey), "")

	if serviceName == "" {
		return "", fmt.Errorf("request ctx metadata missing service name in key %s", ServiceNameKey)
	}

	return serviceName, nil
}

// GetServiceNameFromStream extracts the nitric service name from the incoming context of a grpc stream
func GetServiceNameFromStream(stream grpc.ServerStream) (string, error) {
	return GetServiceNameFromIncomingContext(stream.Context())
}
