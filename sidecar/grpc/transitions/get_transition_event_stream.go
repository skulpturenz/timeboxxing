package transitions

import (
	componentTransitions "github.com/skulpturenz/timeboxxing/sidecar/components/transitions"
	transitionsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/transitions/v1"
	"google.golang.org/grpc"
)

func (s *Server) GetTransitionEventStream(req *transitionsv1.GetTransitionEventsRequest, stream grpc.ServerStreamingServer[transitionsv1.TransitionEvent]) error {
	filters, err := filtersFromRequest(req)
	if err != nil {
		return err
	}

	subscription := s.transitions.Subscribe(componentTransitions.SubscribeParams{Filters: filters})
	defer subscription.Close()

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case event := <-subscription.Events:
			if err := stream.Send(transitionEventToProto(event)); err != nil {
				return err
			}
		}
	}
}
