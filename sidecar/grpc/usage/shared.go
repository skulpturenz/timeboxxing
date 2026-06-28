package usage

import (
	componentUsage "github.com/skulpturenz/timeboxxing/sidecar/components/usage"
	usagev1 "github.com/skulpturenz/timeboxxing/sidecar/gen/usage/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func windowFromRequest(req *usagev1.GetUsageEventsRequest) (componentUsage.Window, error) {
	window := componentUsage.Window{}
	if req.GetStartedAt() == nil {
		return window, status.Error(codes.InvalidArgument, "started_at is required")
	}
	if req.GetEndedAt() == nil {
		return window, status.Error(codes.InvalidArgument, "ended_at is required")
	}
	if err := req.GetStartedAt().CheckValid(); err != nil {
		return window, status.Errorf(codes.InvalidArgument, "started_at is invalid: %v", err)
	}
	if err := req.GetEndedAt().CheckValid(); err != nil {
		return window, status.Errorf(codes.InvalidArgument, "ended_at is invalid: %v", err)
	}

	window.StartedAt = req.GetStartedAt().AsTime()
	window.EndedAt = req.GetEndedAt().AsTime()
	if !window.EndedAt.After(window.StartedAt) {
		return window, status.Error(codes.InvalidArgument, "ended_at must be after started_at")
	}

	return window, nil
}

func eventToProto(event componentUsage.Event) *usagev1.UsageEvent {
	return &usagev1.UsageEvent{
		Id:                    event.ID,
		Title:                 event.Title,
		SourceName:            event.SourceName,
		Source:                sourceToProto(event.Source),
		StartedAt:             timestamppb.New(event.StartedAt),
		EndedAt:               timestamppb.New(event.EndedAt),
		Reason:                event.Reason,
		ApplicationName:       event.ApplicationName,
		CdpUrl:                event.CDPURL,
		ApplicationIdentifier: event.ApplicationIdentifier,
		ApplicationPath:       event.ApplicationPath,
		Active:                event.Active,
	}
}

func sourceToProto(source componentUsage.Source) usagev1.UsageSource {
	switch source {
	case componentUsage.SourceApplication:
		return usagev1.UsageSource_USAGE_SOURCE_APPLICATION
	case componentUsage.SourceBrowser:
		return usagev1.UsageSource_USAGE_SOURCE_BROWSER
	case componentUsage.SourceIdle:
		return usagev1.UsageSource_USAGE_SOURCE_IDLE
	default:
		return usagev1.UsageSource_USAGE_SOURCE_UNSPECIFIED
	}
}
