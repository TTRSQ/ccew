package ccew

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"testing"
)

func TestNewBitflyer(t *testing.T) {
	name := "bitflyer"
	keyFile := "src/bitflyer/key.json"
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

	bf, err := New(name, Key{
		APIKey:    bfKey.APIKey,
		APISecKey: bfKey.APISecKey,
	})
	fmt.Println(bf.ExchangeName(), "ok.")
	if err != nil {
		t.Errorf(" %s\n", err.Error())
	}
}
