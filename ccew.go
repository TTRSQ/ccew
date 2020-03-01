package ccew

import (
	"errors"

	"github.com/TTRSQ/ccew/interface/exchange"
	"github.com/TTRSQ/ccew/src/bitflyer"
)

// this is factory of ccew.

// New .. get exchange wrapper.
func New(exchangeName string, key exchange.Key) (exchange.Exchange, error) {
	if exchangeName == "bitflyer" {
		return bitflyer.New(key)
	}
	return nil, errors.New("exchange name:" + exchangeName + "not found.")
}
