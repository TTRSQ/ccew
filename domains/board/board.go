package board

import "github.com/TTRSQ/ccew/domains/base"

// type Item struct {
// 	order.Id
// 	base.Norm
// }

// Board list of asks and bids.
type Board struct {
	ExchangeName string
	Symbol       string
	MidPrice     float64
	Asks         []base.Norm
	Bids         []base.Norm
}
