package dummy

import (
	"errors"
	"fmt"
	"sort"

	"github.com/TTRSQ/ccew/domains/base"
	"github.com/TTRSQ/ccew/domains/board"
	"github.com/TTRSQ/ccew/domains/order"
	"github.com/TTRSQ/ccew/domains/order/id"
	"github.com/TTRSQ/ccew/domains/stock"
	"github.com/TTRSQ/ccew/interface/exchange"
)

type dummy struct {
	host      string
	name      string
	buyReqs   []boardElm
	sellReqs  []boardElm
	stockSize float64
	incID     int
	ltp       float64
	cash      float64
	takerFee  float64
	makerFee  float64
	bestAsk   float64
	bestBid   float64
}

type boardElm struct {
	ID    string
	Price float64
	Size  float64
}

// New return exchange obj.
func New(key exchange.Key) (exchange.Exchange, error) {
	dm := dummy{}
	dm.name = "dummy"
	dm.host = "ttrsq.com"
	dm.buyReqs = []boardElm{}
	dm.sellReqs = []boardElm{}
	dm.bestAsk = 100000000
	if key.SpecificParam["makerFee"] != nil {
		dm.makerFee = key.SpecificParam["makerFee"].(float64)
	}
	if key.SpecificParam["takerFee"] != nil {
		dm.takerFee = key.SpecificParam["takerFee"].(float64)
	}

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

func (dm *dummy) CreateOrder(price, size float64, isBuy bool, symbol, orderType string) (*order.Responce, error) {
	localID := dm.incrementalID()
	executed := false
	if orderType == "LIMIT" {
		executed = dm.addOrder(isBuy, boardElm{
			ID:    localID,
			Price: price,
			Size:  size,
		})
	} else {
		executed = true
	}
	if executed {
		if isBuy {
			dm.stockSize += size
			dm.cash -= price * size * (1.0 + dm.takerFee)
		} else {
			dm.stockSize -= size
			dm.cash += price * size * (1.0 - dm.takerFee)
		}
	}

	return &order.Responce{
		ID: id.NewID(dm.name, symbol, localID),
		// TODO: using best ask, bid.
		FilledSize: map[bool]float64{true: size, false: 0.0}[orderType == dm.OrderTypes().Market],
	}, nil
}

func (dm *dummy) LiquidationOrder(price, size float64, isBuy bool, symbol, orderType string) (*order.Responce, error) {
	return nil, errors.New("EditOrder not supported. ")
}

func (dm *dummy) EditOrder(symbol, localID string, price, size float64) (*order.Order, error) {
	// キャンセル
	canceled, isBuy := dm.cancelOrder(localID)

	if !canceled {
		return &order.Order{}, fmt.Errorf("order:%s not found. ", localID)
	}

	// 新規作成
	ord, err := dm.CreateOrder(price, size, isBuy, symbol, dm.OrderTypes().Limit)
	if err != nil {
		return nil, err
	}

	return &order.Order{
		ID: ord.ID,
		Request: order.Request{
			Norm: base.Norm{
				Price: price,
				Size:  size - ord.FilledSize,
			},
			Symbol:    symbol,
			IsBuy:     isBuy,
			OrderType: dm.OrderTypes().Limit,
		},
	}, nil
}

func (dm *dummy) CancelOrder(symbol, localID string) error {
	dm.cancelOrder(localID)
	return nil
}

func (dm *dummy) CancelAllOrder(symbol string) error {
	dm.buyReqs = []boardElm{}
	dm.sellReqs = []boardElm{}
	return nil
}

func (dm *dummy) ActiveOrders(symbol string) ([]order.Order, error) {
	ret := []order.Order{}
	for _, data := range dm.buyReqs {
		ret = append(ret, order.Order{
			ID: id.NewID(dm.name, symbol, data.ID),
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
	for _, data := range dm.sellReqs {
		ret = append(ret, order.Order{
			ID: id.NewID(dm.name, symbol, data.ID),
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

	stock := stock.Stock{Symbol: symbol, Summary: dm.stockSize}
	if dm.stockSize > 0 {
		stock.LongSize = dm.stockSize
	} else {
		stock.ShortSize = dm.stockSize
	}
	return stock, nil
}

func (dm *dummy) Balance() ([]base.Balance, error) {
	ret := []base.Balance{
		{
			CurrencyCode: "all",
			Size:         dm.cash + dm.stockSize*dm.ltp,
		},
		{
			CurrencyCode: "fiat",
			Size:         dm.cash,
		}, {
			CurrencyCode: "crypto",
			Size:         dm.stockSize,
		},
	}

	return ret, nil
}

func (dm *dummy) Boards(symbol string) (board.Board, error) {
	return board.Board{}, errors.New("not supported. ")
}

func (dm *dummy) InScheduledMaintenance() bool {
	return false
}

func (dm *dummy) UpdateLTP(lastTimePrice float64) error {
	dm.ltp = lastTimePrice
	dm.updateExecution()
	return nil
}

func (dm *dummy) UpdateBestPrice(bestAsk, bestBid float64) error {
	dm.bestAsk = bestAsk
	dm.bestBid = bestBid
	return nil
}

func (dm *dummy) updateExecution() {
	executedIDs := []string{}
	for _, v := range dm.buyReqs {
		if dm.ltp < v.Price {
			executedIDs = append(executedIDs, v.ID)
			dm.stockSize += v.Size
			dm.cash -= v.Price * v.Size * (1.0 + dm.makerFee)
		}
	}

	for _, v := range dm.sellReqs {
		if dm.ltp > v.Price {
			executedIDs = append(executedIDs, v.ID)
			dm.stockSize -= v.Size
			dm.cash += v.Price * v.Size * (1.0 - dm.makerFee)
		}
	}

	// 不要になったオーダー削除
	for _, v := range executedIDs {
		dm.cancelOrder(v)
	}
}

func (dm *dummy) addOrder(isBuy bool, ele boardElm) bool {
	executed := false
	if isBuy {
		if dm.bestAsk < ele.Price {
			executed = true
			dm.bestAsk *= 1.1
		} else {
			dm.buyReqs = append(dm.buyReqs, ele)
			if len(dm.buyReqs) > 1 {
				sort.Slice(dm.buyReqs, func(i, j int) bool {
					return dm.buyReqs[i].Price > dm.buyReqs[j].Price
				})
			}
		}
	} else {
		if dm.bestBid > ele.Price {
			executed = true
			dm.bestBid /= 1.1
		} else {
			dm.sellReqs = append(dm.sellReqs, ele)
			if len(dm.sellReqs) > 1 {
				sort.Slice(dm.sellReqs, func(i, j int) bool {
					return dm.sellReqs[i].Price < dm.sellReqs[j].Price
				})
			}
		}
	}
	return executed
}

// returns canceled, isbuy
func (dm *dummy) cancelOrder(id string) (bool, bool) {
	newBuyReqs := []boardElm{}
	newSellReqs := []boardElm{}

	canceled := false
	isBuy := false

	for _, v := range dm.buyReqs {
		if id != v.ID {
			newBuyReqs = append(newBuyReqs, v)
		} else {
			canceled = true
			isBuy = true
		}
	}

	for _, v := range dm.sellReqs {
		if id != v.ID {
			newSellReqs = append(newSellReqs, v)
		} else {
			canceled = true
		}
	}

	dm.buyReqs = newBuyReqs
	dm.sellReqs = newSellReqs

	return canceled, isBuy
}

func (dm *dummy) incrementalID() string {
	id := dm.incID
	dm.incID++
	return fmt.Sprint(id)
}
