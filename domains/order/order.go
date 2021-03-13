package order

import (
	"github.com/TTRSQ/ccew/domains/base"
	"github.com/TTRSQ/ccew/domains/order/id"
)

// Request ..
type Request struct {
	base.Norm
	Symbol    string
	IsBuy     bool
	OrderType string
}

// Responce
type Responce struct {
	ID         id.ID
	FilledSize float64
}

// Order OrderObj
type Order struct {
	id.ID
	Request
}
