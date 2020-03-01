package exchange

import (
	"github.com/TTRSQ/ccew/domains/execution"
	"github.com/TTRSQ/ccew/domains/order"
	"github.com/TTRSQ/ccew/domains/stock"
)

// Key .. key data for use private apis.
type Key struct {
	APIKey    string
	APISecKey string
}

// Exchange 取引所のラッパーentity
type Exchange interface {
	// public
	ExchangeName() string
	InScheduledMaintenance() bool

	// private
	CreateOrder(order order.Request) (*order.ID, error)
	CancelOrder(orderID order.ID) error
	CancelAllOrder(symbol string) error
	ActiveOrders(symbol string) ([]order.Order, error)
	Stocks(symbol string) (stock.Stock, error)
}

// Stream socketを起動し受け取る
type Stream interface {
	Start() error
	// Read Execution, error Executionはなかったらnilが飛ぶ
	Read() (execution.Execution, error)
}
