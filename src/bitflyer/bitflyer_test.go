package bitflyer

import (
	"fmt"
	"testing"

	"github.com/TTRSQ/ccew/interface/exchange"
)

func getExchange() exchange.Exchange {
	return New("key.json")
}

func TestStocks(t *testing.T) {
	bf := getExchange()
	st, err := bf.Stocks("FX_BTC_JPY")
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("result: %+v\n", st)
}

func TestActiveOrders(t *testing.T) {
	bf := getExchange()
	orders, err := bf.ActiveOrders("FX_BTC_JPY")
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("result: %+v\n", orders)
}
