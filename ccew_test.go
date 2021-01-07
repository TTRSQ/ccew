package ccew

import (
	"fmt"
	"testing"
)

func TestNewBitflyer(t *testing.T) {
	name := "bitflyer"

	bf, err := New(name, ExchangeKey{
		APIKey:    "hoge",
		APISecKey: "fuga",
	})

	fmt.Println(bf.ExchangeName(), "ok.")
	if err != nil {
		t.Errorf(" %s\n", err.Error())
	}
}
