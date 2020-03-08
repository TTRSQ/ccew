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

type OrderTypes struct {
	Market string
	Limit  string
}

type Symbols struct {
	BtcJpy   string
	FxBtcJpy string
}

// Exchange 取引所のラッパーentity
type Exchange interface {
	// const
	OrderTypes() OrderTypes
	Symbols() Symbols

	// public
	ExchangeName() string
	InScheduledMaintenance() bool

	// private
	CreateOrder(price, size float64, isBuy bool, symbol, orderType string) (*order.ID, error)
	CancelOrder(symbol, localID string) error
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
