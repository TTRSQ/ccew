package ccew

import (
	"testing"
)

func TestNewBitflyer(t *testing.T) {
	name := "bitflyer"

	bf, err := Bitflyer(ExchangeKey{
		APIKey:    "hoge",
		APISecKey: "fuga",
	})

	if err != nil {
		t.Errorf(" %s\n", err.Error())
	}

	if bf.ExchangeName() != name {
		t.Error(bf.ExchangeName() + " != " + name)
	}
}

func TestNewFtx(t *testing.T) {
	name := "ftx"

	ftx, err := Ftx(ExchangeKey{
		APIKey:    "hoge",
		APISecKey: "fuga",
	})

	if err != nil {
		t.Errorf(" %s\n", err.Error())
	}

	if ftx.ExchangeName() != name {
		t.Error(ftx.ExchangeName() + " != " + name)
	}
}
