package grpcx

import (
	"context"

	"github.com/pterm/pterm"
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

func CreateServiceIdInterceptor(serviceName string) (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
			pterm.Info.Printf("%+v\n", ctx)
			// Inject the name of the service
			md, _ := metadata.FromIncomingContext(ctx)
			md.Append(ServiceNameKey, serviceName) // example of adding new metadata

			newCtx := metadata.NewOutgoingContext(ctx, md)
			pterm.Info.Printf("%+v\n", newCtx)
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
