package exchange

import (
	"github.com/TTRSQ/ccew/domains/base"
	"github.com/TTRSQ/ccew/domains/board"
	"github.com/TTRSQ/ccew/domains/execution"
	"github.com/TTRSQ/ccew/domains/order"
	"github.com/TTRSQ/ccew/domains/stock"
)

// Key .. key data for use private apis.
type Key struct {
	APIKey        string
	APISecKey     string
	SpecificParam map[string]interface{}
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

	// public
	ExchangeName() string
	InScheduledMaintenance() bool
	Boards(symbol string) (board.Board, error)

	// private
	CreateOrder(price, size float64, isBuy bool, symbol, orderType string) (*order.Responce, error)
	LiquidationOrder(price, size float64, isBuy bool, symbol, orderType string) (*order.Responce, error)
	EditOrder(symbol, localID string, price, size float64) (*order.Order, error)
	CancelOrder(symbol, localID string) error
	CancelAllOrder(symbol string) error
	ActiveOrders(symbol string) ([]order.Order, error)
	Stocks(symbol string) (stock.Stock, error)
	Balance() ([]base.Balance, error)

	// for backtest
	UpdateLTP(ltp float64) error
	UpdateBestPrice(bestAsk, bestBid float64) error
}

// Stream socketを起動し受け取る
type Stream interface {
	Start() error
	// Read Execution, error Executionはなかったらnilが飛ぶ
	Read() (execution.Execution, error)
}
