package server

import (
	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"

	"github.com/masroorhasan/myapp/tracer"
)

//NewServer gives and GRPC server
func NewServer() (*grpc.Server, error) {
	// initialize tracer
	tracer, closer, err := tracer.NewTracer()
	defer closer.Close()
	if err != nil {
		return &grpc.Server{}, err
	}
	opentracing.SetGlobalTracer(tracer)

	// initialize grpc server with chained interceptors
	s := grpc.NewServer(
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			// add opentracing stream interceptor to chain
			grpc_opentracing.StreamServerInterceptor(grpc_opentracing.WithTracer(tracer)),
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			// add opentracing unary interceptor to chain
			grpc_opentracing.UnaryServerInterceptor(grpc_opentracing.WithTracer(tracer)),
		)),
	)
	return s, nil
}
