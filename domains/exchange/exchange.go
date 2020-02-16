package exchange

import (
	"github.com/TTRSQ/ccew/domains/execution"
	"github.com/TTRSQ/ccew/domains/order"
	"github.com/TTRSQ/ccew/domains/stock"
)

// Exchange 取引所のラッパーentity
type Exchange interface {
	CreateOrder(order order.Order) (*order.ID, error)
	CancelOrder(orderID order.ID) error
	CancelAllOrder(symbol string) error
	ActiveOrders(symbol string) ([]order.Order, error)
	Stocks(symbol string) ([]stock.Stock, error)
	InScheduledMaintenance() bool
	Update(e execution.Execution) error
}

// Stream socketを起動し受け取る
type Stream interface {
	Start() error
	// Read Execution, error Executionはなかったらnilが飛ぶ
	Read() (execution.Execution, error)
}
