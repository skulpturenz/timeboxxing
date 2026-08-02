package timeline

import (
	"container/list"
	"slices"

	"github.com/negrel/assert"
	"github.com/skulpturenz/timeboxxing/sidecar/components/timeline/models"
)

func assertSorted(stream []models.ForegroundProcess) {
	assert.Condition(func() bool { // expect asc sort
		cloned := slices.Clone(stream)
		slices.SortFunc(cloned, func(x models.ForegroundProcess, y models.ForegroundProcess) int {
			return x.Timestamp.Compare(y.Timestamp) // asc
		})

		return slices.Equal(stream, cloned)
	})
}

func CollectTimeline(stream []models.ForegroundProcess) *list.List {
	list := list.New()

	assertSorted(stream)

	for _, v := range stream {
		prev := list.Back()
		if prev == nil {
			list.PushBack(v)
			continue
		}

		previous, ok := prev.Value.(models.ForegroundProcess)
		assert.True(ok)

		if !ok || !previous.IsEqual(v) {
			list.PushBack(v)
		}
	}

	return list
}

func SeqTimeline(timeline *list.List) []models.UsageSeq {
	seq := []models.UsageSeq{}

	for e := timeline.Front(); e != nil; e = e.Next() {
		prev := e.Prev()
		next := e.Next()

		var start *models.ForegroundProcess
		if prev != nil {
			v, ok := prev.Value.(models.ForegroundProcess)
			assert.True(ok)

			start = &v
		}

		var end *models.ForegroundProcess
		if next != nil {
			v, ok := next.Value.(models.ForegroundProcess)
			assert.True(ok)

			end = &v
		}

		current, ok := e.Value.(models.ForegroundProcess)
		assert.True(ok)

		item := models.UsageSeq{
			ID:     0, // only when foreground process is from db
			Start:  start,
			End:    end,
			Killed: current.Killed,
		}

		seq = append(seq, item)
	}

	return seq
}
