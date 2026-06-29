package transitions

import (
	componentTransitions "github.com/skulpturenz/timeboxxing/sidecar/components/transitions"
	transitionsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/transitions/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func filtersFromRequest(req *transitionsv1.GetTransitionEventsRequest) (componentTransitions.Filters, error) {
	filters := componentTransitions.Filters{}
	if req.GetStartedAt() != nil {
		if err := req.GetStartedAt().CheckValid(); err != nil {
			return filters, status.Errorf(codes.InvalidArgument, "started_at is invalid: %v", err)
		}
		startedAt := req.GetStartedAt().AsTime().UTC()
		filters.StartedAt = &startedAt
	}
	if req.GetEndedAt() != nil {
		if err := req.GetEndedAt().CheckValid(); err != nil {
			return filters, status.Errorf(codes.InvalidArgument, "ended_at is invalid: %v", err)
		}
		endedAt := req.GetEndedAt().AsTime().UTC()
		filters.EndedAt = &endedAt
	}

	return filters, nil
}

func transitionEventToProto(event componentTransitions.Event) *transitionsv1.TransitionEvent {
	return &transitionsv1.TransitionEvent{
		Id:              event.ID,
		ApplicationName: event.ApplicationName,
		Reason:          event.Reason,
		StartedAt:       timestamppb.New(event.StartedAt.UTC()),
		EndedAt:         timestamppb.New(event.EndedAt.UTC()),
		Browser:         event.Browser,
		Tab:             event.Tab,
		Idle:            event.Idle,
		CdpUrl:          event.CDPURL,
		Pid:             event.PID,
	}
}
