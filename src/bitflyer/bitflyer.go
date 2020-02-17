package bitflyer

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/TTRSQ/ccew/exchange"
	"github.com/TTRSQ/ccew/domains/execution"
	"github.com/TTRSQ/ccew/domains/order"
	"github.com/TTRSQ/ccew/domains/base"
	"github.com/TTRSQ/ccew/domains/stock"
)

type bitflyer struct {
	apiKey    string
	apiSecKey string
	host      string
	name      string
}

type key struct {
	APIKey    string `json:"api_key"`
	APISecKey string `json:"api_sec_key"`
}

// New return exchange obj.
func New() exchange.Exchange {
	bf := bitflyer{}

	bytes, err := ioutil.ReadFile("../adopter/apiClient/bitflyer/key.json")
	if err != nil {
		log.Fatal(err)
	}
	bfKey := key{}
	json.Unmarshal(bytes, &bfKey)

	bf.name = "bitflyer"
	bf.host = "api.bitflyer.com"
	bf.apiKey = bfKey.APIKey
	bf.apiSecKey = bfKey.APISecKey

	return &bf
}

func (bf *bitflyer) CreateOrder(o order.Request) (order.ID, error) {
	// リクエスト
	type Req struct {
		ProductCode    string  `json:"product_code"`
		ChildOrderType string  `json:"child_order_type"`
		Side           string  `json:"side"`
		Price          int     `json:"price"`
		Size           float64 `json:"size"`
		MinuteToExpire int     `json:"minute_to_expire"`
		TimeInForce    string  `json:"time_in_force"`
	}
	res, _ := bf.postRequest("/v1/me/sendchildorder", Req{
		ProductCode:    o.Symbol,
		ChildOrderType: o.Type,
		Side:           map[bool]string{true: "BUY", false: "SELL"}[o.IsBuy],
		Price:          int(o.Price),
		Size:           o.Size,
		MinuteToExpire: 10000,
		TimeInForce:    "GTC",
	})

	// レスポンスの変換
	type Res struct {
		ID           string `json:"child_order_acceptance_id"`
		ErrorMessage string `json:"error_message"`
	}
	resData := Res{}
	json.Unmarshal(res, &resData)
	if resData.ErrorMessage != "" {
		return order.ID{}, errors.New(resData.ErrorMessage)
	}

	return order.ID{ExchangeName: bf.name, LocalID: resData.ID}, nil
}

func (bf *bitflyer) CancelOrder(id order.ID, symbol string) error {
	type Req struct {
		ProductCode            string `json:"product_code"`
		ChildOrderAcceptanceID string `json:"child_order_acceptance_id"`
	}

	_, err := bf.postRequest("/v1/me/cancelchildorder", Req{
		ProductCode:            symbol,
		ChildOrderAcceptanceID: order.LocalID,
	})
	return err
}

func (bf *bitflyer) CancelAllOrder(symbol string) error {
	type Req struct {
		ProductCode            string `json:"product_code"`
		ChildOrderAcceptanceID string `json:"child_order_acceptance_id"`
	}

	_, err := bf.postRequest("/v1/me/cancelallchildorders", Req{
		ProductCode: symbol,
	})
	return err
}

func (bf *bitflyer) ActiveOrders(symbol string) ([]order.Order, error) {
	type Req struct {
		ChildOrderState string `json:"child_order_state"`
		Symbol          string `json:"product_code"`
	}
	res, _ := bf.getRequest("/v1/me/getchildorders", Req{
		ChildOrderState: "ACTIVE",
		Symbol:          symbol,
	})

	// レスポンスの変換
	type Res struct {
		ID                     int     `json:"id"`
		ChildOrderID           string  `json:"child_order_id"`
		ProductCode            string  `json:"product_code"`
		Side                   string  `json:"side"`
		ChildOrderType         string  `json:"child_order_type"`
		Price                  int     `json:"price"`
		AveragePrice           int     `json:"average_price"`
		Size                   float64 `json:"size"`
		ChildOrderState        string  `json:"child_order_state"`
		ExpireDate             string  `json:"expire_date"`
		ChildOrderDate         string  `json:"child_order_date"`
		ChildOrderAcceptanceID string  `json:"child_order_acceptance_id"`
		OutstandingSize        int     `json:"outstanding_size"`
		CancelSize             int     `json:"cancel_size"`
		ExecutedSize           float64 `json:"executed_size"`
		TotalCommission        int     `json:"total_commission"`
	}
	resData := []Res{}
	json.Unmarshal(res, &resData)

	// 返却値の作成
	ret := []order.Order{}
	for _, data := range resData {
		//log.Printf("%+v\n", data)
		ret = append(ret, order.Order{
			ID:    order.NewID(bf.name, data.ProductCode, data.ChildOrderAcceptanceID),
			IsBuy: data.Side == "BUY",
			Type:  data.ChildOrderType,
			Norm: base.Norm{
				Price: float64(data.Price),
				Size:  data.Size,
			},
		})
	}

	return ret, nil
}

func (bf *bitflyer) Stocks(symbol string) ([]stock.Stock, error) {
	type Req struct {
		Symbol string `json:"product_code"`
	}
	res, _ := bf.getRequest("/v1/me/getpositions", Req{
		Symbol: symbol,
	})

	// レスポンスの変換
	type Res struct {
		ProductCode         string  `json:"product_code"`
		Side                string  `json:"side"`
		Price               float64 `json:"price"`
		Size                float64 `json:"size"`
		Commission          float64 `json:"commission"`
		SwapPointAccumulate float64 `json:"swap_point_accumulate"`
		RequireCollateral   float64 `json:"require_collateral"`
		OpenDate            string  `json:"open_date"`
		Leverage            float64 `json:"leverage"`
		Pnl                 float64 `json:"pnl"`
		Sfd                 float64 `json:"sfd"`
	}
	resData := []Res{}
	json.Unmarshal(res, &resData)

	// 返却値の作成
	ret := []stock.Stock{}
	for _, data := range resData {
		//log.Printf("%+v\n", data)
		ret = append(ret, stock.Stock{
			Symbol: data.ProductCode,
			IsBuy:  data.Side == "BUY",
			Price:  data.Price,
			Size:   data.Size,
		})
	}

	return ret, nil
}

// func (bf *bitflyer) InScheduledMaintenance() bool {
// 	jst := utiltime.Jst()
// 	// 355 <= time <= 415で落とす
// 	from := 355
// 	to := 415
// 	now := jst.Hour()*100 + jst.Minute()
// 	return from <= now && now <= to
// }

func (bf *bitflyer) getRequest(path string, param interface{}) ([]byte, error) {
	// jsonをガリガリする
	jsonParam, _ := json.Marshal(param)
	jsonStr := strings.Trim(string(jsonParam), "{}")
	jstrs := strings.Split(jsonStr, ",")
	uparm := url.Values{}
	for _, v := range jstrs {
		vv := strings.Split(v, ":")
		uparm.Add(strings.Trim(vv[0], "\""), strings.Trim(vv[1], "\""))
	}

	uri := url.URL{Scheme: "https", Host: bf.host, Path: path, RawQuery: uparm.Encode()}
	timestamp := string(time.Now().Unix())
	req, _ := http.NewRequest(
		"GET",
		uri.String(),
		nil,
	)

	req.Header.Add("ACCESS-KEY", bf.apiKey)
	req.Header.Add("ACCESS-TIMESTAMP", timestamp)

	rawSign := timestamp + "GET" + path + "?" + uparm.Encode()
	sign := bf.makeHMAC(rawSign)

	req.Header.Add("ACCESS-SIGN", sign)

	return bf.request(req)
}

func (bf *bitflyer) postRequest(path string, param interface{}) ([]byte, error) {
	url := url.URL{Scheme: "https", Host: bf.host, Path: path}
	timestamp := string(time.Now().Unix())
	jsonParam, _ := json.Marshal(param)
	req, _ := http.NewRequest(
		"POST",
		url.String(),
		bytes.NewBuffer(jsonParam),
	)

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("ACCESS-KEY", bf.apiKey)
	req.Header.Add("ACCESS-TIMESTAMP", timestamp)

	rawSign := timestamp + "POST" + path + string(jsonParam)
	sign := bf.makeHMAC(rawSign)

	req.Header.Add("ACCESS-SIGN", sign)

	return bf.request(req)
}

func (bf *bitflyer) makeHMAC(msg string) string {
	mac := hmac.New(sha256.New, []byte(bf.apiSecKey))
	mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil))
}

func (bf *bitflyer) request(req *http.Request) ([]byte, error) {
	client := new(http.Client)
	resp, err := client.Do(req)

	if err != nil {
		log.Fatalf("err ==> %+v\nreq ==> %v\n", err, req)
	}
	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		log.Printf("body ==> %s\n", string(body))
		log.Fatalf("resp ==> %+v\nreq ==> %v\n", resp, req)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	return body, err
}

