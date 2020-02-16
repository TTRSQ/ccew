package order

import (
	"github.com/TTRSQ/ccew/domains/base/norm"
	"github.com/TTRSQ/ccew/domains/order/id"
)

// ID order id
type ID id.ID

// NewID .. make id obj.
func NewID(exchange, symbol, localID string) *ID {
	return &ID{
		ExchangeName: exchange,
		Symbol:       symbol,
		LocalID:      localID,
	}
}

// Order OrderObj
type Order struct {
	ID
	norm.Norm
	IsBuy     bool
	OrderType string
}
