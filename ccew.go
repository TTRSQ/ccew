package ccew

import (
	"errors"

	"github.com/TTRSQ/ccew/interface/exchange"
	"github.com/TTRSQ/ccew/src/bitflyer"
)

// this is factory of ccew.

// Key apikey and apiseckey.
type Key struct {
	APIKey    string
	APISecKey string
}

// New .. get exchange wrapper.
func New(exchangeName string, key Key) (exchange.Exchange, error) {
	if exchangeName == "bitflyer" {
		return bitflyer.New(translateKey(key))
	}
	return nil, errors.New("exchange name:" + exchangeName + "not found.")
}

func translateKey(key Key) exchange.Key {
	return exchange.Key{
		APIKey:    key.APIKey,
		APISecKey: key.APISecKey,
	}
}
