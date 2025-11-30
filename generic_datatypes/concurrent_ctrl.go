// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
package generic_datatypes

import (
	"sync/atomic"
)

type ConcurrentCtrl struct {
	Ch   chan struct{}
	Flag atomic.Bool
}
