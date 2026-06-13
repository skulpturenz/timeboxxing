package helloservicetwo

import hellov1 "github.com/skulpturenz/timeboxxing/sidecar/gen/hello/v1"

type HelloServiceTwoServer struct {
	hellov1.UnimplementedHelloServiceTwoServer
}
