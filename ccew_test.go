package ccew

import (
	"testing"
	"fmt"
)

func TestNew(t *testing.T) {
	bf, err := New("bitflyer")
	fmt.Println(bf.ExchangeName(), "ok.")
	if err != nil {
		t.Errorf(" %s\n", err.Error())
	}
}