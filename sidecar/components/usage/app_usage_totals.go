package usage

import (
	"context"
	"sort"
	"strings"
	"time"
)

type AppUsageTotalsParams struct {
	Window      Window
	Limit       int
	IncludeIdle bool
}

type AppUsageTotals struct {
	StartedAt     time.Time
	EndedAt       time.Time
	TotalDuration time.Duration
	Buckets       []AppUsageBucket
}

type AppUsageBucket struct {
	Name                  string
	Source                Source
	Duration              time.Duration
	SessionCount          int
	ApplicationIdentifier string
	ApplicationPath       string
}

func (s *Service) GetAppUsageTotals(ctx context.Context, params AppUsageTotalsParams) (AppUsageTotals, error) {
	events, err := s.GetEvents(ctx, GetEventsParams{Window: params.Window})
	if err != nil {
		return AppUsageTotals{}, err
	}

	type aggregate struct {
		bucket AppUsageBucket
	}
	byName := map[string]*aggregate{}
	var total time.Duration
	for _, event := range events {
		if event.Source == SourceIdle && !params.IncludeIdle {
			continue
		}
		duration := clippedUsageDuration(event.StartedAt, event.EndedAt, params.Window.StartedAt, params.Window.EndedAt)
		if duration <= 0 {
			continue
		}

		name := appUsageName(event)
		item := byName[name]
		if item == nil {
			item = &aggregate{
				bucket: AppUsageBucket{
					Name:                  name,
					Source:                event.Source,
					ApplicationIdentifier: strings.TrimSpace(event.ApplicationIdentifier),
					ApplicationPath:       strings.TrimSpace(event.ApplicationPath),
				},
			}
			byName[name] = item
		}
		item.bucket.Duration += duration
		item.bucket.SessionCount++
		if item.bucket.ApplicationIdentifier == "" {
			item.bucket.ApplicationIdentifier = strings.TrimSpace(event.ApplicationIdentifier)
		}
		if item.bucket.ApplicationPath == "" {
			item.bucket.ApplicationPath = strings.TrimSpace(event.ApplicationPath)
		}
		total += duration
	}

	buckets := make([]AppUsageBucket, 0, len(byName))
	for _, item := range byName {
		buckets = append(buckets, item.bucket)
	}
	sort.Slice(buckets, func(i, j int) bool {
		if buckets[i].Duration == buckets[j].Duration {
			return buckets[i].Name < buckets[j].Name
		}
		return buckets[i].Duration > buckets[j].Duration
	})
	if params.Limit > 0 && len(buckets) > params.Limit {
		buckets = buckets[:params.Limit]
	}

	return AppUsageTotals{
		StartedAt:     params.Window.StartedAt,
		EndedAt:       params.Window.EndedAt,
		TotalDuration: total,
		Buckets:       buckets,
	}, nil
}

func appUsageName(event Event) string {
	if event.Source == SourceIdle {
		return "Idle"
	}
	if name := strings.TrimSpace(event.ApplicationName); name != "" {
		return name
	}
	if name := strings.TrimSpace(event.SourceName); name != "" && name != "Application" && name != "Browser" {
		return name
	}
	return "Unknown application"
}

func clippedUsageDuration(start, end, windowStart, windowEnd time.Time) time.Duration {
	if start.Before(windowStart) {
		start = windowStart
	}
	if end.After(windowEnd) {
		end = windowEnd
	}
	if !end.After(start) {
		return 0
	}
	return end.Sub(start)
}
