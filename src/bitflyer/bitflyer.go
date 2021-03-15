package bitflyer

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/TTRSQ/ccew/domains/base"
	"github.com/TTRSQ/ccew/domains/board"
	"github.com/TTRSQ/ccew/domains/order"
	"github.com/TTRSQ/ccew/domains/order/id"
	"github.com/TTRSQ/ccew/domains/stock"
	"github.com/TTRSQ/ccew/interface/exchange"
)

type bitflyer struct {
	apiKey    string
	apiSecKey string
	host      string
	name      string
}

// New return exchange obj.
func New(key exchange.Key) (exchange.Exchange, error) {
	bf := bitflyer{}
	bf.name = "bitflyer"
	bf.host = "api.bitflyer.com"

	if key.APIKey == "" || key.APISecKey == "" {
		return nil, errors.New("APIKey and APISecKey Required")
	}
	bf.apiKey = key.APIKey
	bf.apiSecKey = key.APISecKey

	return &bf, nil
}

func (bf *bitflyer) ExchangeName() string {
	return bf.name
}

func (bf *bitflyer) OrderTypes() exchange.OrderTypes {
	return exchange.OrderTypes{
		Limit:  "LIMIT",
		Market: "MARKET",
	}
}

func (bf *bitflyer) CreateOrder(price, size float64, isBuy bool, symbol, orderType string) (*order.Responce, error) {
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
	res, err := bf.postRequest("/v1/me/sendchildorder", Req{
		ProductCode:    symbol,
		ChildOrderType: orderType,
		Side:           map[bool]string{true: "BUY", false: "SELL"}[isBuy],
		Price:          int(price),
		Size:           size,
		MinuteToExpire: 10000,
		TimeInForce:    "GTC",
	})
	if err != nil {
		return nil, err
	}

	// レスポンスの変換
	type Res struct {
		ID           string `json:"child_order_acceptance_id"`
		ErrorMessage string `json:"error_message"`
	}
	resData := Res{}
	json.Unmarshal(res, &resData)
	if resData.ErrorMessage != "" {
		return nil, errors.New(resData.ErrorMessage)
	}

	return &order.Responce{
		ID: id.NewID(bf.name, symbol, resData.ID),
		// 成り行きであればすべて約定する前提
		FilledSize: 0.0,
	}, nil
}

func (bf *bitflyer) CancelOrder(symbol, localID string) error {
	type Req struct {
		ProductCode            string `json:"product_code"`
		ChildOrderAcceptanceID string `json:"child_order_acceptance_id"`
	}

	_, err := bf.postRequest("/v1/me/cancelchildorder", Req{
		ProductCode:            symbol,
		ChildOrderAcceptanceID: localID,
	})
	return err
}

func (bf *bitflyer) EditOrder(symbol, localID string, price, size float64) (*order.Order, error) {
	return nil, errors.New("EditOrder not supported.")
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
	res, err := bf.getRequest("/v1/me/getchildorders", Req{
		ChildOrderState: "ACTIVE",
		Symbol:          symbol,
	})
	if err != nil {
		return nil, err
	}

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
			ID: id.NewID(bf.name, data.ProductCode, data.ChildOrderAcceptanceID),
			Request: order.Request{
				IsBuy:     data.Side == "BUY",
				OrderType: data.ChildOrderType,
				Norm: base.Norm{
					Price: float64(data.Price),
					Size:  data.Size,
				},
			},
		})
	}
	return ret, nil
}

func (bf *bitflyer) Stocks(symbol string) (stock.Stock, error) {
	type Req struct {
		Symbol string `json:"product_code"`
	}
	res, err := bf.getRequest("/v1/me/getpositions", Req{
		Symbol: symbol,
	})
	if err != nil {
		return stock.Stock{}, err
	}

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
	ret := stock.Stock{Symbol: symbol}
	for _, data := range resData {
		if data.Side == "SELL" {
			ret.Size -= data.Size
		} else {
			ret.Size += data.Size
		}
	}

	return ret, nil
}

func (bf *bitflyer) Balance() ([]base.Balance, error) {
	type Req struct {
		Symbol string `json:"product_code"`
	}
	res, err := bf.getRequest("/v1/me/getbalance", Req{})
	if err != nil {
		return nil, err
	}

	// レスポンスの変換
	type Res []struct {
		CurrencyCode string  `json:"currency_code"`
		Amount       float64 `json:"amount"`
		Available    int     `json:"available"`
	}
	resData := Res{}
	json.Unmarshal(res, &resData)

	// 返却値の作成
	ret := []base.Balance{}
	for _, data := range resData {
		ret = append(ret, base.Balance{
			CurrencyCode: data.CurrencyCode,
			Size:         data.Amount,
		})
	}

	return ret, nil
}

func (bf *bitflyer) Boards(symbol string) (board.Board, error) {
	type Req struct {
		Symbol string `json:"product_code"`
	}
	res, err := bf.getRequest("/v1/getboard", Req{
		Symbol: symbol,
	})
	if err != nil {
		return board.Board{}, err
	}

	// レスポンスの変換
	type Res struct {
		MidPrice float64 `json:"mid_price"`
		Bids     []struct {
			Price float64 `json:"price"`
			Size  float64 `json:"size"`
		} `json:"bids"`
		Asks []struct {
			Price float64 `json:"price"`
			Size  float64 `json:"size"`
		} `json:"asks"`
	}
	resData := Res{}
	json.Unmarshal(res, &resData)

	// 返却値の作成
	asks := []base.Norm{}
	for _, v := range resData.Asks {
		asks = append(asks, base.Norm{
			Price: v.Price,
			Size:  v.Size,
		})
	}
	bids := []base.Norm{}
	for _, v := range resData.Bids {
		bids = append(bids, base.Norm{
			Price: v.Price,
			Size:  v.Size,
		})
	}

	return board.Board{
		Symbol:   symbol,
		MidPrice: resData.MidPrice,
		Asks:     asks,
		Bids:     bids,
	}, nil
}

func (bf *bitflyer) InScheduledMaintenance() bool {
	// jst := utiltime.Jst()
	// // 355 <= time <= 415で落とす
	// from := 355
	// to := 415
	// now := jst.Hour()*100 + jst.Minute()
	// return from <= now && now <= to
	return false
}

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
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
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
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
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
		errStr := ""
		errStr += fmt.Sprintf("body ==> %s\n", string(body))
		errStr += fmt.Sprintf("resp ==> %+v\nreq ==> %v\n", resp, req)
		return nil, errors.New(errStr)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	return body, err
}

func (bf *bitflyer) UpdateLTP(lastTimePrice float64) error {
	return errors.New("not supported.")
}
