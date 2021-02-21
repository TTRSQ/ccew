package bitflyer

import (
	"fmt"
	"testing"

	"github.com/TTRSQ/ccew/interface/exchange"
)

func getExchange() exchange.Exchange {
	bf, _ := New(exchange.Key{
		APIKey:    "hoge",
		APISecKey: "hoge",
	})
	return bf
}

func TestBoards(t *testing.T) {
	bf := getExchange()
	board, err := bf.Boards("FX_BTC_JPY")
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("result:%+v\n", board)
}
