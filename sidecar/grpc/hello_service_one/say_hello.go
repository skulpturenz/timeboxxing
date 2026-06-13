package helloserviceone

import (
	"context"

	hellov1 "github.com/skulpturenz/timeboxxing/sidecar/gen/hello/v1"
)

func (HelloServiceOneServer) SayHello(_ context.Context, req *hellov1.HelloRequest) (*hellov1.HelloResponse, error) {
	name := req.GetName()
	if name == "" {
		name = "world"
	}

	return &hellov1.HelloResponse{Message: "hello from service one, " + name}, nil
}
