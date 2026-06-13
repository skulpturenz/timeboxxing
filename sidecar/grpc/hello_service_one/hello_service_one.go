package helloserviceone

import hellov1 "github.com/skulpturenz/timeboxxing/sidecar/gen/hello/v1"

type HelloServiceOneServer struct {
	hellov1.UnimplementedHelloServiceOneServer
}
