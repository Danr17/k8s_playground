package client

import (
	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	"google.golang.org/grpc"

	"github.com/masroorhasan/myapp/tracer"
)

//NewClientConn initilize the grpc connection
func NewClientConn(address string) (*grpc.ClientConn, error) {
	// initialize tracer
	tracer, closer, err := tracer.NewTracer()
	defer closer.Close()
	if err != nil {
		return &grpc.ClientConn{}, err
	}

	// initialize client with tracing interceptor using grpc client side chaining
	return grpc.Dial(
		address,
		grpc.WithStreamInterceptor(grpc_middleware.ChainStreamClient(
			grpc_opentracing.StreamClientInterceptor(grpc_opentracing.WithTracer(tracer)),
		)),
		grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(
			grpc_opentracing.UnaryClientInterceptor(grpc_opentracing.WithTracer(tracer)),
		)),
	)
}
