package bitflyer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"testing"

	"github.com/TTRSQ/ccew/interface/exchange"
)

func getExchange() exchange.Exchange {
	keyFile := "key.json"
	type key struct {
		APIKey    string `json:"api_key"`
		APISecKey string `json:"api_sec_key"`
	}
	bfKey := key{}
	bytes, err := ioutil.ReadFile(keyFile)
	if err != nil {
		log.Fatal(err)
	}
	json.Unmarshal(bytes, &bfKey)

	bf, _ := New(exchange.Key{
		APIKey:    bfKey.APIKey,
		APISecKey: bfKey.APISecKey,
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
