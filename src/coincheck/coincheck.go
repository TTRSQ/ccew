package coincheck

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
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/TTRSQ/ccew/domains/base"
	"github.com/TTRSQ/ccew/domains/board"
	"github.com/TTRSQ/ccew/domains/order"
	"github.com/TTRSQ/ccew/domains/order/id"
	"github.com/TTRSQ/ccew/domains/stock"
	"github.com/TTRSQ/ccew/interface/exchange"
)

type keyStruct struct {
	id  string
	sec string
}

type coincheck struct {
	keys   []keyStruct
	host   string
	name   string
	keyIdx int
}

// New return exchange obj.
func New(key exchange.Key) (exchange.Exchange, error) {
	cc := coincheck{}
	cc.name = "coincheck"
	cc.host = "coincheck.com"

	if key.APIKey == "" || key.APISecKey == "" {
		return nil, errors.New("APIKey and APISecKey Required")
	}
	cc.keys = []keyStruct{
		{
			id:  key.APIKey,
			sec: key.APISecKey,
		},
	}

	// nonceError回避用のkeyを追加する
	_, exist := key.SpecificParam["additional_keys"]
	if exist {
		additionalKeys := key.SpecificParam["additional_keys"].([][]string)
		for i := range additionalKeys {
			cc.keys = append(cc.keys, keyStruct{
				id:  additionalKeys[i][0],
				sec: additionalKeys[i][1],
			})
		}
	}

	return &cc, nil
}

func (cc *coincheck) ExchangeName() string {
	return cc.name
}

func (cc *coincheck) OrderTypes() exchange.OrderTypes {
	return exchange.OrderTypes{
		Limit:  "limit",
		Market: "market",
	}
}

func (cc *coincheck) CreateOrder(price, size float64, isBuy bool, symbol, orderType string) (*order.Responce, error) {
	// リクエスト
	type Req struct {
		OrderType string      `json:"order_type"`
		Pair      string      `json:"pair"`
		Size      float64     `json:"amount"`
		Price     interface{} `json:"rate"`
	}
	oType := map[bool]string{true: "buy", false: "sell"}[isBuy]
	if orderType == cc.OrderTypes().Market {
		oType = "market_" + oType
	}
	res, err := cc.postRequest("/api/exchange/orders", &Req{
		Pair:      symbol,
		OrderType: oType,
		Price:     map[bool]interface{}{true: price, false: nil}[orderType == cc.OrderTypes().Limit],
		Size:      size,
	})
	if err != nil {
		return nil, err
	}

	// レスポンスの変換
	type Res struct {
		Success      bool        `json:"success"`
		ID           int         `json:"id"`
		Rate         string      `json:"rate"`
		Amount       string      `json:"amount"`
		OrderType    string      `json:"order_type"`
		StopLossRate interface{} `json:"stop_loss_rate"`
		Pair         string      `json:"pair"`
		CreatedAt    time.Time   `json:"created_at"`
		Error        string      `json:"error"`
	}
	resData := Res{}
	json.Unmarshal(res, &resData)
	if !resData.Success {
		return nil, errors.New(resData.Error)
	}
	return &order.Responce{
		ID:         id.NewID(cc.name, symbol, fmt.Sprint(resData.ID)),
		FilledSize: 0,
	}, nil
}

func (cc *coincheck) LiquidationOrder(price, size float64, isBuy bool, symbol, orderType string) (*order.Responce, error) {
	return nil, errors.New("LiquidationOrder: not supported.")
}

func (cc *coincheck) EditOrder(symbol, localID string, price, size float64) (*order.Order, error) {
	return nil, errors.New("EditOrder: not supported.")
}

func (cc *coincheck) CancelOrder(symbol, localID string) error {
	_, err := cc.deleteRequest("/api/exchange/orders/"+localID, nil)
	return err
}

func (cc *coincheck) CancelAllOrder(symbol string) error {
	return errors.New("CancelAllOrder: not supported.")
}

func (cc *coincheck) ActiveOrders(symbol string) ([]order.Order, error) {
	type Req struct {
		Symbol     int    `json:"product_id"`
		OrderState string `json:"status"`
	}
	res, err := cc.getRequest("/api/exchange/orders/opens", nil)
	if err != nil {
		return nil, err
	}

	// レスポンスの変換
	type Res struct {
		Success bool `json:"success"`
		Orders  []struct {
			ID                     int         `json:"id"`
			OrderType              string      `json:"order_type"`
			Rate                   float64     `json:"rate"`
			Pair                   string      `json:"pair"`
			PendingAmount          string      `json:"pending_amount"`
			PendingMarketBuyAmount interface{} `json:"pending_market_buy_amount"`
			StopLossRate           interface{} `json:"stop_loss_rate"`
			CreatedAt              time.Time   `json:"created_at"`
		} `json:"orders"`
		Error string `json:"error"`
	}
	resData := Res{}
	json.Unmarshal(res, &resData)
	if !resData.Success {
		return nil, errors.New(resData.Error)
	}

	// 返却値の作成
	ret := []order.Order{}
	for _, data := range resData.Orders {
		price := data.Rate
		size, _ := strconv.ParseFloat(data.PendingAmount, 64)
		ret = append(ret, order.Order{
			ID: id.NewID(cc.name, data.Pair, fmt.Sprint(data.ID)),
			Request: order.Request{
				IsBuy:     data.OrderType == "buy",
				OrderType: data.OrderType,
				Norm: base.Norm{
					Price: price,
					Size:  size,
				},
			},
		})
	}
	return ret, nil
}

func (cc *coincheck) Stocks(symbol string) (stock.Stock, error) {
	ret := stock.Stock{Symbol: symbol}
	return ret, errors.New("Stocks: not supported.")
}

func (cc *coincheck) Balance() ([]base.Balance, error) {
	res, err := cc.getRequest("/api/accounts/balance", nil)
	if err != nil {
		return nil, err
	}

	// レスポンスの変換
	type Res struct {
		Success      bool   `json:"success"`
		Jpy          string `json:"jpy"`
		Btc          string `json:"btc"`
		JpyReserved  string `json:"jpy_reserved"`
		BtcReserved  string `json:"btc_reserved"`
		JpyLendInUse string `json:"jpy_lend_in_use"`
		BtcLendInUse string `json:"btc_lend_in_use"`
		JpyLent      string `json:"jpy_lent"`
		BtcLent      string `json:"btc_lent"`
		JpyDebt      string `json:"jpy_debt"`
		BtcDebt      string `json:"btc_debt"`
		Error        string `json:"error"`
	}
	resData := Res{}
	json.Unmarshal(res, &resData)
	if !resData.Success {
		return nil, errors.New(resData.Error)
	}
	// 返却値の作成
	jpySize, _ := strconv.ParseFloat(resData.Jpy, 64)
	jpyReserved, _ := strconv.ParseFloat(resData.JpyReserved, 64)
	ret := []base.Balance{{
		CurrencyCode: "jpy",
		Size:         jpySize + jpyReserved,
	}}
	btcSize, _ := strconv.ParseFloat(resData.Btc, 64)
	btcReserved, _ := strconv.ParseFloat(resData.BtcReserved, 64)
	ret = append(ret, base.Balance{
		CurrencyCode: "btc",
		Size:         btcSize + btcReserved,
	})

	return ret, nil
}

func (cc *coincheck) Boards(symbol string) (board.Board, error) {
	type Req struct {
		Symbol string `json:"pair"`
	}
	res, err := cc.getRequest("/api/order_books", &Req{
		Symbol: symbol,
	})
	if err != nil {
		return board.Board{}, err
	}

	// レスポンスの変換
	type Res struct {
		Asks  [][]interface{} `json:"asks"`
		Bids  [][]interface{} `json:"bids"`
		Error string          `json:"error"`
	}
	resData := Res{}
	json.Unmarshal(res, &resData)

	// 返却値の作成
	asks := []base.Norm{}
	for _, v := range resData.Asks {
		price, _ := strconv.ParseFloat(v[0].(string), 64)
		size, _ := strconv.ParseFloat(v[1].(string), 64)
		asks = append(asks, base.Norm{
			Price: price,
			Size:  size,
		})
	}
	bids := []base.Norm{}
	for _, v := range resData.Bids {
		price, _ := strconv.ParseFloat(v[0].(string), 64)
		size, _ := strconv.ParseFloat(v[1].(string), 64)
		bids = append(bids, base.Norm{
			Price: price,
			Size:  size,
		})
	}

	return board.Board{
		Symbol:   symbol,
		MidPrice: (asks[0].Price + bids[0].Price) / 2,
		Asks:     asks,
		Bids:     bids,
	}, nil
}

func (cc *coincheck) InScheduledMaintenance() bool {
	return false
}

func (cc *coincheck) getRequest(path string, param interface{}) ([]byte, error) {
	query := ""
	if param != nil {
		query = structToQuery(param)
	}

	u := url.URL{Scheme: "https", Host: cc.host, Path: path, RawQuery: query}
	req, _ := http.NewRequest(
		"GET",
		u.String(),
		nil,
	)

	req.Header.Add("Content-Type", "application/json")

	timeStamp := fmt.Sprintf("%d", time.Now().UnixNano())
	sig, apiKey := cc.makeHMAC(timeStamp + u.String())
	req.Header.Add("ACCESS-KEY", apiKey)
	req.Header.Add("ACCESS-SIGNATURE", sig)
	req.Header.Add("ACCESS-NONCE", timeStamp)

	return cc.request(req)
}

func (cc *coincheck) deleteRequest(path string, param interface{}) ([]byte, error) {
	u := url.URL{Scheme: "https", Host: cc.host, Path: path}
	jsonParam, _ := json.Marshal(param)
	req, _ := http.NewRequest(
		"DELETE",
		u.String(),
		bytes.NewBuffer(jsonParam),
	)

	req.Header.Add("Content-Type", "application/json")

	timeStamp := fmt.Sprintf("%d", time.Now().UnixNano())
	sig, apiKey := cc.makeHMAC(timeStamp + u.String() + string(jsonParam))
	req.Header.Add("ACCESS-KEY", apiKey)
	req.Header.Add("ACCESS-SIGNATURE", sig)
	req.Header.Add("ACCESS-NONCE", timeStamp)

	return cc.request(req)
}

func (cc *coincheck) postRequest(path string, param interface{}) ([]byte, error) {
	u := url.URL{Scheme: "https", Host: cc.host, Path: path}
	jsonParam, _ := json.Marshal(param)
	req, _ := http.NewRequest(
		"POST",
		u.String(),
		bytes.NewBuffer(jsonParam),
	)

	req.Header.Add("Content-Type", "application/json")

	timeStamp := fmt.Sprintf("%d", time.Now().UnixNano())
	sig, apiKey := cc.makeHMAC(timeStamp + u.String() + string(jsonParam))
	req.Header.Add("ACCESS-KEY", apiKey)
	req.Header.Add("ACCESS-SIGNATURE", sig)
	req.Header.Add("ACCESS-NONCE", timeStamp)

	return cc.request(req)
}

func structToQuery(data interface{}) string {
	elem := reflect.ValueOf(data).Elem()
	size := elem.NumField()

	queries := []string{}
	for i := 0; i < size; i++ {
		value := elem.Field(i).Interface()
		field := elem.Type().Field(i).Tag.Get("json")
		if fmt.Sprint(value) != "<nil>" {
			switch value.(type) {
			case float64:
				queries = append(queries, field+"="+fmt.Sprintf("%f", value))
			default:
				queries = append(queries, field+"="+fmt.Sprint(value))
			}
		}
	}

	return strings.Join(queries, "&")
}

func (cc *coincheck) makeHMAC(msg string) (string, string) {
	key := cc.getKey()
	mac := hmac.New(sha256.New, []byte(key.sec))
	mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil)), key.id
}

func (cc *coincheck) getKey() keyStruct {
	idx := cc.keyIdx
	cc.keyIdx = (cc.keyIdx + 1) % len(cc.keys)
	return cc.keys[idx]
}

func (cc *coincheck) request(req *http.Request) ([]byte, error) {
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

func (cc *coincheck) UpdateLTP(lastTimePrice float64) error {
	return errors.New("not supported.")
}
