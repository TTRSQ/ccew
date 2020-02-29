package ccew

import (
	"github.com/TTRSQ/ccew/interface/exchange"
	"github.com/TTRSQ/ccew/src/bitflyer"
	"errors"
)

// this is factory of ccew.

// New .. get exchange wrapper.
func New(exchangeName string) (exchange.Exchange, error) {
	if exchangeName == "bitflyer" {
		return bitflyer.New(), nil
	}
	return nil, errors.New("exchange name:" + exchangeName + "not found.")
}
