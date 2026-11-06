package hermes

import (
	"context"
	"fmt"
	"sync"

	"github.com/skulpturenz/timeboxxing/sidecar/utils"
)

type Message = map[any]any

type MiddlewareHandlerFunc = func(sink string, domain string, msg Message, filtered bool)
type Middleware = func(next MiddlewareHandlerFunc) MiddlewareHandlerFunc

type sink struct {
	key     string
	outChan chan<- Message
	filters []func(Message) bool
}

type source struct {
	sourceChan <-chan Message
}

type Hermes struct {
	sources     map[string][]source
	sinks       map[string][]sink
	domains     map[string]chan Message
	once        sync.Once
	initialized bool
	middleware  []Middleware
}

func (h *Hermes) Use(middleware Middleware) *Hermes {
	if h.initialized {
		panic("already initialized")
	}

	h.middleware = append(h.middleware, middleware)

	return h
}

func (h *Hermes) AddSource(domain string, src <-chan Message) {
	if h.initialized {
		panic("already initialized")
	}

	srcs := []source{}
	if h.sources[domain] != nil {
		srcs = append(srcs, h.sources[domain]...)
	}

	srcs = append(srcs, source{sourceChan: src})
	h.sources[domain] = srcs
}

func (h *Hermes) AddSink(domain string, key string, outChan chan<- Message, filters ...func(Message) bool) {
	if h.initialized {
		panic("already initialized")
	}

	sinks := []sink{}
	if h.sources[domain] != nil {
		sinks = append(sinks, h.sinks[domain]...)
	}

	sinks = append(sinks, sink{key: key, outChan: outChan, filters: filters})
	h.sinks[domain] = sinks
}

func (h *Hermes) Plumb(ctx context.Context, domain string) {
	h.once.Do(func() {
		createMiddlewareStack := func() MiddlewareHandlerFunc {
			noop := func(sink string, domain string, msg Message, filtered bool) {}

			if len(h.middleware) == 0 {
				return noop
			}

			combined := noop

			for _, m := range h.middleware {
				combined = m(combined)
			}

			return combined
		}

		middleware := createMiddlewareStack()

		var outChan chan Message
		if h.domains[domain] != nil {
			outChan = h.domains[domain]
		} else {
			outChan = make(chan Message)
		}
		h.domains[domain] = outChan

		srcs := []source{}
		if h.sources[domain] != nil {
			srcs = append(srcs, h.sources[domain]...)
		}

		inChans := []<-chan Message{}
		for _, src := range srcs {
			inChans = append(inChans, src.sourceChan)
		}

		sinks := []sink{}
		if h.sinks[domain] != nil {
			sinks = append(sinks, h.sinks[domain]...)
		}

		outChans := []chan<- Message{}
		for _, sink := range sinks {
			inChan := make(chan Message)

			go func() {
				defer close(inChan)

				seenItems := map[string]int{}

				for {
					select {
					case <-ctx.Done():
						return
					case item, ok := <-inChan:
						if !ok {
							return
						}

						filtered := false
						for _, filter := range sink.filters {
							if !filter(item) {
								filtered = true
								break
							}
						}
						middleware(sink.key, domain, item, filtered)
						if filtered {
							continue
						}

						select {
						case sink.outChan <- item:
							delete(seenItems, fmt.Sprintf("%v", item))
						default:
							seenItems[fmt.Sprintf("%v", item)] += 1
							// will apply backpressure. goroutine blocks until outChan frees up
							inChan <- item
						}
					}
				}
			}()

			outChans = append(outChans, inChan)
		}

		utils.FanInChan(ctx, outChan, inChans...)
		utils.FanOutChan(ctx, outChan, outChans...)

		h.initialized = true
	})
}
