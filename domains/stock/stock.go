package stock

import (
	"github.com/TTRSQ/ccew/domains/base"
)

// Stock StockObj
type Stock struct {
	Symbol string
	IsBuy  bool
	base.Norm
}
