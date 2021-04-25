package bitbank

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

type bitbank struct {
	keys    []keyStruct
	host    string
	pubHost string
	name    string
	keyIdx  int
}

// New return exchange obj.
func New(key exchange.Key) (exchange.Exchange, error) {
	bb := bitbank{}
	bb.name = "bitbank"
	bb.host = "api.bitbank.cc"
	bb.pubHost = "public.bitbank.cc"

	if key.APIKey == "" || key.APISecKey == "" {
		return nil, errors.New("APIKey and APISecKey Required")
	}
	bb.keys = []keyStruct{
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
			bb.keys = append(bb.keys, keyStruct{
				id:  additionalKeys[i][0],
				sec: additionalKeys[i][1],
			})
		}
	}

	return &bb, nil
}

func (bb *bitbank) ExchangeName() string {
	return bb.name
}

func (bb *bitbank) OrderTypes() exchange.OrderTypes {
	return exchange.OrderTypes{
		Limit:  "limit",
		Market: "market",
	}
}

func (bb *bitbank) CreateOrder(price, size float64, isBuy bool, symbol, orderType string) (*order.Responce, error) {
	// リクエスト
	type Req struct {
		Pair   string `json:"pair"`
		Price  string `json:"price"`
		Amount string `json:"amount"`
		Side   string `json:"side"`
		Type   string `json:"type"`
	}
	res, err := bb.postRequest("/v1/user/spot/order", &Req{
		Pair:   symbol,
		Price:  fmt.Sprintf("%f", price),
		Amount: fmt.Sprintf("%f", size),
		Side:   map[bool]string{true: "buy", false: "sell"}[isBuy],
		Type:   orderType,
	})
	if err != nil {
		return nil, err
	}

	// レスポンスの変換
	type Res struct {
		Success int `json:"success"`
		Data    struct {
			OrderID         int    `json:"order_id"`
			Pair            string `json:"pair"`
			Side            string `json:"side"`
			Type            string `json:"type"`
			StartAmount     string `json:"start_amount"`
			RemainingAmount string `json:"remaining_amount"`
			ExecutedAmount  string `json:"executed_amount"`
			Price           string `json:"price"`
			PostOnly        bool   `json:"post_only"`
			AveragePrice    string `json:"average_price"`
			OrderedAt       int    `json:"ordered_at"`
			ExpireAt        int    `json:"expire_at"`
			Status          string `json:"status"`
			Code            string `json:"code"`
		} `json:"data"`
	}
	resData := Res{}
	json.Unmarshal(res, &resData)
	if resData.Success == 0 {
		return nil, errors.New(resData.Data.Code)
	}
	filledSize, _ := strconv.ParseFloat(resData.Data.ExecutedAmount, 64)
	return &order.Responce{
		ID:         id.NewID(bb.name, symbol, fmt.Sprint(resData.Data.OrderID)),
		FilledSize: filledSize,
	}, nil
}

func (bb *bitbank) LiquidationOrder(price, size float64, isBuy bool, symbol, orderType string) (*order.Responce, error) {
	return nil, errors.New("LiquidationOrder: not supported.")
}

func (bb *bitbank) EditOrder(symbol, localID string, price, size float64) (*order.Order, error) {
	return nil, errors.New("EditOrder: not supported.")
}

func (bb *bitbank) CancelOrder(symbol, localID string) error {
	type Req struct {
		Pair    string `json:"pair"`
		OrderID int    `json:"order_id"`
	}
	localIDINT, _ := strconv.ParseInt(localID, 10, 64)
	_, err := bb.postRequest("/v1/user/spot/cancel_order", &Req{
		Pair:    symbol,
		OrderID: int(localIDINT),
	})
	return err
}

func (bb *bitbank) CancelAllOrder(symbol string) error {
	return errors.New("CancelAllOrder: not supported.")
}

func (bb *bitbank) ActiveOrders(symbol string) ([]order.Order, error) {
	type Req struct {
		Symbol string `json:"pair"`
	}
	res, err := bb.getRequest("/v1/user/spot/active_orders", &Req{
		Symbol: symbol,
	}, false)
	if err != nil {
		return nil, err
	}

	// レスポンスの変換
	type Res struct {
		Success int `json:"success"`
		Data    struct {
			Orders []struct {
				OrderID         int    `json:"order_id"`
				Pair            string `json:"pair"`
				Side            string `json:"side"`
				Type            string `json:"type"`
				StartAmount     string `json:"start_amount"`
				RemainingAmount string `json:"remaining_amount"`
				ExecutedAmount  string `json:"executed_amount"`
				Price           string `json:"price"`
				PostOnly        bool   `json:"post_only"`
				AveragePrice    string `json:"average_price"`
				OrderedAt       int    `json:"ordered_at"`
				ExpireAt        int    `json:"expire_at"`
				Status          string `json:"status"`
			} `json:"orders"`
			Code string `json:"code"`
		} `json:"data"`
	}
	resData := Res{}
	json.Unmarshal(res, &resData)
	if resData.Success == 0 {
		return nil, errors.New(resData.Data.Code)
	}

	// 返却値の作成
	ret := []order.Order{}
	for _, data := range resData.Data.Orders {
		price, _ := strconv.ParseFloat(data.Price, 64)
		size, _ := strconv.ParseFloat(data.RemainingAmount, 64)
		ret = append(ret, order.Order{
			ID: id.NewID(bb.name, data.Pair, fmt.Sprint(data.OrderID)),
			Request: order.Request{
				IsBuy:     data.Side == "buy",
				OrderType: data.Type,
				Norm: base.Norm{
					Price: price,
					Size:  size,
				},
			},
			UpdatedAtUnix: data.OrderedAt,
		})
	}
	return ret, nil
}

func (bb *bitbank) Stocks(symbol string) (stock.Stock, error) {
	ret := stock.Stock{Symbol: symbol}
	return ret, errors.New("Stocks: not supported.")
}

func (bb *bitbank) Balance() ([]base.Balance, error) {
	res, err := bb.getRequest("/v1/user/assets", nil, false)
	if err != nil {
		return nil, err
	}

	// レスポンスの変換
	type Res struct {
		Success int `json:"success"`
		Data    struct {
			Assets []struct {
				Asset           string `json:"asset"`
				FreeAmount      string `json:"free_amount"`
				AmountPrecision int    `json:"amount_precision"`
				OnhandAmount    string `json:"onhand_amount"`
				LockedAmount    string `json:"locked_amount"`
				WithdrawalFee   string `json:"withdrawal_fee"`
				StopDeposit     bool   `json:"stop_deposit"`
				StopWithdrawal  bool   `json:"stop_withdrawal"`
			} `json:"assets"`
			Code string `json:"code"`
		} `json:"data"`
	}
	resData := Res{}
	json.Unmarshal(res, &resData)
	if resData.Success == 0 {
		return nil, errors.New(resData.Data.Code)
	}
	// 返却値の作成
	balances := []base.Balance{}
	for _, v := range resData.Data.Assets {
		free, _ := strconv.ParseFloat(v.FreeAmount, 64)
		locked, _ := strconv.ParseFloat(v.LockedAmount, 64)
		balances = append(balances, base.Balance{
			CurrencyCode: v.Asset,
			Size:         free + locked,
		})
	}

	return balances, nil
}

func (bb *bitbank) Boards(symbol string) (board.Board, error) {
	type Req struct {
		Symbol string `json:"pair"`
	}
	res, err := bb.getRequest(symbol+"/depth", nil, true)
	if err != nil {
		return board.Board{}, err
	}

	// レスポンスの変換
	type Res struct {
		Success int `json:"success"`
		Data    struct {
			Asks [][]string `json:"asks"`
			Bids [][]string `json:"bids"`
		} `json:"data"`
	}
	resData := Res{}
	json.Unmarshal(res, &resData)

	// 返却値の作成
	asks := []base.Norm{}
	for _, v := range resData.Data.Asks {
		price, _ := strconv.ParseFloat(v[0], 64)
		size, _ := strconv.ParseFloat(v[1], 64)
		asks = append(asks, base.Norm{
			Price: price,
			Size:  size,
		})
	}
	bids := []base.Norm{}
	for _, v := range resData.Data.Bids {
		price, _ := strconv.ParseFloat(v[0], 64)
		size, _ := strconv.ParseFloat(v[1], 64)
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

func (bb *bitbank) InScheduledMaintenance() bool {
	return false
}

func (bb *bitbank) getRequest(path string, param interface{}, isPublic bool) ([]byte, error) {
	query := ""
	if param != nil {
		query = structToQuery(param)
	}

	host := bb.host
	if isPublic {
		host = bb.pubHost
	}

	u := url.URL{Scheme: "https", Host: host, Path: path, RawQuery: query}
	req, _ := http.NewRequest(
		"GET",
		u.String(),
		nil,
	)

	req.Header.Add("Content-Type", "application/json")

	if !isPublic {
		timeStamp := fmt.Sprintf("%d", time.Now().UnixNano())
		sigPATH := u.Path
		if u.RawQuery != "" {
			sigPATH += "?" + u.RawQuery
		}
		sig, apiKey := bb.makeHMAC(timeStamp + sigPATH)
		req.Header.Add("ACCESS-KEY", apiKey)
		req.Header.Add("ACCESS-SIGNATURE", sig)
		req.Header.Add("ACCESS-NONCE", timeStamp)
	}

	return bb.request(req)
}

func (bb *bitbank) postRequest(path string, param interface{}) ([]byte, error) {
	u := url.URL{Scheme: "https", Host: bb.host, Path: path}
	jsonParam, _ := json.Marshal(param)
	req, _ := http.NewRequest(
		"POST",
		u.String(),
		bytes.NewBuffer(jsonParam),
	)

	req.Header.Add("Content-Type", "application/json")

	timeStamp := fmt.Sprintf("%d", time.Now().UnixNano())
	sig, apiKey := bb.makeHMAC(timeStamp + string(jsonParam))
	req.Header.Add("ACCESS-KEY", apiKey)
	req.Header.Add("ACCESS-SIGNATURE", sig)
	req.Header.Add("ACCESS-NONCE", timeStamp)

	return bb.request(req)
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

func (bb *bitbank) makeHMAC(msg string) (string, string) {
	key := bb.getKey()
	mac := hmac.New(sha256.New, []byte(key.sec))
	mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil)), key.id
}

func (bb *bitbank) getKey() keyStruct {
	idx := bb.keyIdx
	bb.keyIdx = (bb.keyIdx + 1) % len(bb.keys)
	return bb.keys[idx]
}

func (bb *bitbank) request(req *http.Request) ([]byte, error) {
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

func (bb *bitbank) UpdateLTP(lastTimePrice float64) error {
	return errors.New("not supported.")
}
