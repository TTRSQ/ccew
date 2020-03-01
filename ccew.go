package ccew

import (
	"errors"

	"github.com/TTRSQ/ccew/interface/exchange"
	"github.com/TTRSQ/ccew/src/bitflyer"
)

// this is factory of ccew.

// New .. get exchange wrapper.
func New(exchangeName string) (exchange.Exchange, error) {
	if exchangeName == "bitflyer" {
		return bitflyer.New("src/bitflyer/key.json"), nil
	}
	return nil, errors.New("exchange name:" + exchangeName + "not found.")
}
