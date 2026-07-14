package sessionnew

import (
	"context"

	"github.com/jonoton/go-ringbuffer"
)

type Reporter interface {
	From(ctx context.Context, stream *ringbuffer.RingBuffer[ForegroundProcess]) Reporter
	Publish(incoming ForegroundProcess)
}
