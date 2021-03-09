package dummy

import (
	"errors"
	"fmt"
	"sort"

	"github.com/TTRSQ/ccew/domains/base"
	"github.com/TTRSQ/ccew/domains/board"
	"github.com/TTRSQ/ccew/domains/order"
	"github.com/TTRSQ/ccew/domains/stock"
	"github.com/TTRSQ/ccew/interface/exchange"
)

type dummy struct {
	apiKey    string
	apiSecKey string
	host      string
	name      string
}

type boardElm struct {
	ID    string
	Price float64
	Size  float64
}

var buyReqs []boardElm
var sellReqs []boardElm
var stockSize float64
var incID int
var ltp float64
var cash float64

func init() {
	buyReqs = []boardElm{}
	sellReqs = []boardElm{}
	stockSize = 0
	incID = 0
	cash = 0
}

// New return exchange obj.
func New(key exchange.Key) (exchange.Exchange, error) {
	dm := dummy{}
	dm.name = "dummy"
	dm.host = "ttrsq.com"

	return &dm, nil
}

func (dm *dummy) ExchangeName() string {
	return dm.name
}

func (dm *dummy) OrderTypes() exchange.OrderTypes {
	return exchange.OrderTypes{
		Limit:  "LIMIT",
		Market: "MARKET",
	}
}

func (dm *dummy) CreateOrder(price, size float64, isBuy bool, symbol, orderType string) (*order.ID, error) {
	localID := incrementalID()
	executed := false
	if orderType == "LIMIT" {
		executed = addOrder(isBuy, boardElm{
			ID:    localID,
			Price: price,
			Size:  size,
		})
	} else {
		executed = true
	}
	if executed {
		if isBuy {
			stockSize += size
			cash -= ltp * size
		} else {
			stockSize -= size
			cash += ltp * size
		}
	}
	return &order.ID{ExchangeName: dm.name, LocalID: localID}, nil
}

func (dm *dummy) CancelOrder(symbol, localID string) error {
	cancelOrder(localID)
	return nil
}

func (dm *dummy) CancelAllOrder(symbol string) error {
	buyReqs = []boardElm{}
	sellReqs = []boardElm{}
	return nil
}

func (dm *dummy) ActiveOrders(symbol string) ([]order.Order, error) {
	ret := []order.Order{}
	for _, data := range buyReqs {
		ret = append(ret, order.Order{
			ID: order.ID{ExchangeName: dm.name, LocalID: data.ID},
			Request: order.Request{
				IsBuy:     true,
				OrderType: "LIMIT",
				Norm: base.Norm{
					Price: data.Price,
					Size:  data.Size,
				},
			},
		})
	}
	for _, data := range sellReqs {
		ret = append(ret, order.Order{
			ID: order.ID{ExchangeName: dm.name, LocalID: data.ID},
			Request: order.Request{
				IsBuy:     false,
				OrderType: "LIMIT",
				Norm: base.Norm{
					Price: data.Price,
					Size:  data.Size,
				},
			},
		})
	}
	return ret, nil
}

func (dm *dummy) Stocks(symbol string) (stock.Stock, error) {
	return stock.Stock{Symbol: symbol, Size: stockSize}, nil
}

func (dm *dummy) Balance() ([]base.Balance, error) {
	ret := []base.Balance{{
		CurrencyCode: "fiat",
		Size:         cash + stockSize*ltp,
	}}

	return ret, nil
}

func (dm *dummy) Boards(symbol string) (board.Board, error) {
	return board.Board{}, errors.New("not supported.")
}

func (dm *dummy) InScheduledMaintenance() bool {
	return false
}

func (dm *dummy) UpdateLTP(lastTimePrice float64) error {
	ltp = lastTimePrice
	return nil
}

func updateExecution() {
	executedIDs := []string{}
	for _, v := range buyReqs {
		if ltp < v.Price {
			executedIDs = append(executedIDs, v.ID)
			stockSize += v.Size
			cash -= ltp * v.Size
		}
	}

	for _, v := range sellReqs {
		if ltp > v.Price {
			executedIDs = append(executedIDs, v.ID)
			stockSize -= v.Size
			cash += ltp * v.Size
		}
	}

	// 不要になったオーダー削除
	for _, v := range executedIDs {
		cancelOrder(v)
	}
}

func addOrder(isBuy bool, ele boardElm) bool {
	executed := false
	if isBuy {
		if ltp < ele.Price {
			executed = true
		} else {
			buyReqs = append(buyReqs, ele)
			if len(buyReqs) > 1 {
				sort.Slice(buyReqs, func(i, j int) bool {
					return buyReqs[i].Price > buyReqs[j].Price
				})
			}
		}
	} else {
		if ltp > ele.Price {
			executed = true
		} else {
			sellReqs = append(sellReqs, ele)
			if len(sellReqs) > 1 {
				sort.Slice(sellReqs, func(i, j int) bool {
					return sellReqs[i].Price < sellReqs[j].Price
				})
			}
		}
	}
	return executed
}

func cancelOrder(id string) {
	newBuyReqs := []boardElm{}
	newSellReqs := []boardElm{}

	for _, v := range buyReqs {
		if id != v.ID {
			newBuyReqs = append(newBuyReqs, v)
		}
	}

	for _, v := range sellReqs {
		if id != v.ID {
			newSellReqs = append(newSellReqs, v)
		}
	}

	buyReqs = newBuyReqs
	sellReqs = newSellReqs
}

func incrementalID() string {
	id := incID
	incID++
	return fmt.Sprint(id)
}
