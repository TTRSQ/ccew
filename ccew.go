package ccew

import (
	"github.com/TTRSQ/ccew/interface/exchange"
	"github.com/TTRSQ/ccew/src/bitflyer"
	"github.com/TTRSQ/ccew/src/ftx"
)

// ExchangeKey ..
type ExchangeKey = exchange.Key

// this is factory of ccew.

// Bitflyer .. no SpecificParam.
func Bitflyer(key exchange.Key) (exchange.Exchange, error) {
	return bitflyer.New(key)
}

// Ftx .. SpecificParam [FTX-SUBACCOUNT(string)]
func Ftx(key exchange.Key) (exchange.Exchange, error) {
	return ftx.New(key)
}
