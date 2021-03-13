package execution

import (
	"time"

	"github.com/TTRSQ/ccew/domains/base"
	"github.com/TTRSQ/ccew/domains/order/id"
)

// Execution information of someone's excution.
type Execution struct {
	id.ID
	base.Norm
	IsBuy     bool
	OccuredAt time.Time
}
