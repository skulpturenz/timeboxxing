package usage

import (
	"strings"

	timelinemodels "github.com/skulpturenz/timeboxxing/sidecar/components/timeline/models"
	usagev1 "github.com/skulpturenz/timeboxxing/sidecar/gen/usage/v1"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// timelinePageSize is how many entries each keyset page of a timeline read carries.
const timelinePageSize = 256

func windowFromRequest(req *usagev1.GetUsageEventsRequest) (utils.TimeSpan, error) {
	window := utils.TimeSpan{}
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

	window[0] = req.GetStartedAt().AsTime().UTC()
	window[1] = req.GetEndedAt().AsTime().UTC()
	if !window[1].After(window[0]) {
		return window, status.Error(codes.InvalidArgument, "ended_at must be after started_at")
	}

	return window, nil
}

// usageSeqToProto projects one timeline entry onto a usage event. Every field is read from the
// entry's opening observation — the closing one only supplies the end instant, because it already
// describes whatever was focused next. previous is the entry immediately before this one and only
// feeds the reason.
//
// It reports false for an entry that is still open or falls outside the window: an entry that has
// not ended is not a recorded stretch.
func usageSeqToProto(seq timelinemodels.UsageSeq, previous *timelinemodels.UsageSeq, window utils.TimeSpan) (*usagev1.UsageEvent, bool) {
	if seq.Start == nil || seq.End == nil {
		return nil, false
	}

	span := seq.Span()
	if !span.Overlaps(window) {
		return nil, false
	}

	start := seq.Start
	event := &usagev1.UsageEvent{
		Id:              seq.ID,
		Title:           start.Title(),
		SourceName:      start.SourceName(),
		Source:          sourceToProto(*start),
		StartedAt:       timestamppb.New(span[0].UTC()),
		EndedAt:         timestamppb.New(span[1].UTC()),
		Reason:          timelinemodels.ReasonFor(seq, previous).String(),
		ApplicationName: start.ApplicationName(),
		CdpUrl:          start.Enrichments.Browser.CdpURL,
		Pid:             int32(utils.Coalesce(start.PID, 0)),
	}

	// an idle stretch is not attributable to an application, so it reports no identity to key on
	if !start.Idle {
		event.ApplicationIdentifier = strings.TrimSpace(utils.Coalesce(start.AppIdentifier, ""))
		event.ApplicationPath = strings.TrimSpace(utils.Coalesce(start.AppPath, ""))
	}

	return event, true
}

func sourceToProto(foregroundProcess timelinemodels.ForegroundProcess) usagev1.UsageSource {
	switch {
	case foregroundProcess.Idle:
		return usagev1.UsageSource_USAGE_SOURCE_IDLE
	case foregroundProcess.IsBrowser():
		return usagev1.UsageSource_USAGE_SOURCE_BROWSER
	default:
		return usagev1.UsageSource_USAGE_SOURCE_APPLICATION
	}
}
