package datatypes

import "sync/atomic"

type ConcurrentCtrl struct {
	Ch   chan struct{}
	Flag atomic.Bool
}
