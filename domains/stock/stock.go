package stock

import (
	"github.com/TTRSQ/ccew/domains/base"
)

// Stcok StockObj
type Stock struct {
	Symbol string
	IsBuy  bool
	base.Norm
}
