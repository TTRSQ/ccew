package ccew

import (
	"github.com/TTRSQ/ccew/interface/exchange"
	"github.com/TTRSQ/ccew/src/bitbank"
	"github.com/TTRSQ/ccew/src/bitflyer"
	"github.com/TTRSQ/ccew/src/bybit"
	"github.com/TTRSQ/ccew/src/coincheck"
	"github.com/TTRSQ/ccew/src/dummy"
	"github.com/TTRSQ/ccew/src/ftx"
	"github.com/TTRSQ/ccew/src/gmo"
	"github.com/TTRSQ/ccew/src/liquid"
)

// ExchangeKey ..
type ExchangeKey = exchange.Key

// this is factory of ccew.

// Bitflyer .. no SpecificParam.
func Bitflyer(key exchange.Key) (exchange.Exchange, error) {
	return bitflyer.New(key)
}

// Ftx .. SpecificParam FTX-SUBACCOUNT : string
func Ftx(key exchange.Key) (exchange.Exchange, error) {
	return ftx.New(key)
}

// ByBit .. no SpecificParam.
func ByBit(key exchange.Key) (exchange.Exchange, error) {
	return bybit.New(key)
}

// BitBank .. SpecificParam additional_keys : [][]string{ [id, sec],[id, sec].. }
func BitBank(key exchange.Key) (exchange.Exchange, error) {
	return bitbank.New(key)
}

// Liquid .. SpecificParam additional_keys : [][]string{ [id, sec],[id, sec].. }
func Liquid(key exchange.Key) (exchange.Exchange, error) {
	return liquid.New(key)
}

// CoinCheck .. SpecificParam additional_keys : [][]string{ [id, sec],[id, sec].. }
func CoinCheck(key exchange.Key) (exchange.Exchange, error) {
	return coincheck.New(key)
}

// Gmo .. SpecificParam proxyURL : *url.URL
func Gmo(key exchange.Key) (exchange.Exchange, error) {
	return gmo.New(key)
}

// Dummy .. no SpecificParam.
func Dummy(key exchange.Key) (exchange.Exchange, error) {
	return dummy.New(key)
}
