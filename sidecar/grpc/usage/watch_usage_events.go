package usage

import (
	componentUsage "github.com/skulpturenz/timeboxxing/sidecar/components/usage"
	usagev1 "github.com/skulpturenz/timeboxxing/sidecar/gen/usage/v1"
	"google.golang.org/grpc"
)

func (s *Server) WatchUsageEvents(req *usagev1.GetUsageEventsRequest, stream grpc.ServerStreamingServer[usagev1.UsageEvent]) error {
	window, err := windowFromRequest(req)
	if err != nil {
		return err
	}

	subscription := s.usage.Subscribe(stream.Context(), componentUsage.SubscribeParams{Window: window})
	defer subscription.Close()

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case event, ok := <-subscription.Events:
			if !ok {
				return nil
			}
			if err := stream.Send(eventToProto(event)); err != nil {
				return err
			}
		}
	}
}
