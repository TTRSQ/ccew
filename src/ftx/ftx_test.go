package ftx

import (
	"fmt"
	"testing"

	"github.com/TTRSQ/ccew/interface/exchange"
)

func getExchange() exchange.Exchange {
	ftx, _ := New(exchange.Key{
		APIKey:    "hoge",
		APISecKey: "hoge",
	})
	return ftx
}

func TestBoards(t *testing.T) {
	ftx := getExchange()
	board, err := ftx.Boards("BTC-PERP")
	if err != nil {
		t.Error("error:", err)
	}
	fmt.Printf("result:%+v\n", board)
}
