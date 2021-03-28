package gmo

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

type gmo struct {
	key  keyStruct
	host string
	name string
}

// New return exchange obj.
func New(key exchange.Key) (exchange.Exchange, error) {
	gmo := gmo{}
	gmo.name = "gmo"
	gmo.host = "api.coin.z.com"

	if key.APIKey == "" || key.APISecKey == "" {
		return nil, errors.New("APIKey and APISecKey Required")
	}
	gmo.key = keyStruct{
		id:  key.APIKey,
		sec: key.APISecKey,
	}

	return &gmo, nil
}

func (gmo *gmo) ExchangeName() string {
	return gmo.name
}

func (gmo *gmo) OrderTypes() exchange.OrderTypes {
	return exchange.OrderTypes{
		Limit:  "LIMIT",
		Market: "MARKET",
	}
}

func (gmo *gmo) CreateOrder(price, size float64, isBuy bool, symbol, orderType string) (*order.Responce, error) {
	// リクエスト
	type Req struct {
		Symbol        string      `json:"symbol"`
		Side          string      `json:"side"`
		ExecutionType string      `json:"executionType"`
		Price         interface{} `json:"price"`
		Size          string      `json:"size"`
	}
	res, err := gmo.postRequest("/private/v1/order", &Req{
		Symbol:        symbol,
		Side:          map[bool]string{true: "BUY", false: "SELL"}[isBuy],
		ExecutionType: orderType,
		Price:         map[bool]interface{}{true: fmt.Sprint(int(price + 0.5)), false: nil}[orderType == gmo.OrderTypes().Limit],
		Size:          fmt.Sprint(size),
	})
	if err != nil {
		return nil, err
	}

	// レスポンスの変換
	type Res struct {
		Status       int       `json:"status"`
		ID           string    `json:"data"`
		Responsetime time.Time `json:"responsetime"`
		Messages     []struct {
			MessageCode   string `json:"message_code"`
			MessageString string `json:"message_string"`
		} `json:"messages"`
	}
	resData := Res{}
	json.Unmarshal(res, &resData)
	if resData.Status != 0 {
		return nil, errors.New(fmt.Sprintf("%+v", resData.Messages))
	}
	return &order.Responce{
		ID:         id.NewID(gmo.name, symbol, fmt.Sprint(resData.ID)),
		FilledSize: 0,
	}, nil
}

func (gmo *gmo) EditOrder(symbol, localID string, price, size float64) (*order.Order, error) {
	// リクエスト
	type Req struct {
		OrderID      int    `json:"orderId"`
		Price        string `json:"price"`
		LosscutPrice string `json:"losscutPrice"`
	}
	idInt, _ := strconv.ParseInt(localID, 10, 64)
	res, err := gmo.postRequest("/private/v1/changeOrder", &Req{
		OrderID: int(idInt),
		Price:   fmt.Sprint(int(price + 0.5)),
	})
	if err != nil {
		return nil, err
	}

	// レスポンスの変換
	type Res struct {
		Status       int       `json:"status"`
		Responsetime time.Time `json:"responsetime"`
		Messages     []struct {
			MessageCode   string `json:"message_code"`
			MessageString string `json:"message_string"`
		} `json:"messages"`
	}
	resData := Res{}
	json.Unmarshal(res, &resData)
	if resData.Status != 0 {
		return nil, errors.New(fmt.Sprintf("%+v", resData.Messages))
	}

	return &order.Order{
		ID:      id.NewID(gmo.name, symbol, localID),
		Request: order.Request{},
	}, nil
}

func (gmo *gmo) CancelOrder(symbol, localID string) error {
	// リクエスト
	type Req struct {
		OrderID int `json:"orderId"`
	}
	idInt, _ := strconv.ParseInt(localID, 10, 64)
	res, err := gmo.postRequest("/private/v1/cancelOrder", &Req{
		OrderID: int(idInt),
	})

	// レスポンスの変換
	type Res struct {
		Status       int       `json:"status"`
		Responsetime time.Time `json:"responsetime"`
		Messages     []struct {
			MessageCode   string `json:"message_code"`
			MessageString string `json:"message_string"`
		} `json:"messages"`
	}
	resData := Res{}
	json.Unmarshal(res, &resData)
	if resData.Status != 0 {
		return errors.New(fmt.Sprintf("%+v", resData.Messages))
	}

	return err
}

func (gmo *gmo) CancelAllOrder(symbol string) error {

	return errors.New("not supported.")
}

func (gmo *gmo) ActiveOrders(symbol string) ([]order.Order, error) {
	type Req struct {
		Symbol string `json:"symbol"`
	}
	res, err := gmo.getRequest("/private/v1/activeOrders", &Req{
		Symbol: symbol,
	})
	if err != nil {
		return nil, err
	}

	// レスポンスの変換
	type Res struct {
		Status int `json:"status"`
		Data   struct {
			Pagination struct {
				CurrentPage int `json:"currentPage"`
				Count       int `json:"count"`
			} `json:"pagination"`
			List []struct {
				RootOrderID   int       `json:"rootOrderId"`
				OrderID       int       `json:"orderId"`
				Symbol        string    `json:"symbol"`
				Side          string    `json:"side"`
				OrderType     string    `json:"orderType"`
				ExecutionType string    `json:"executionType"`
				SettleType    string    `json:"settleType"`
				Size          string    `json:"size"`
				ExecutedSize  string    `json:"executedSize"`
				Price         string    `json:"price"`
				LosscutPrice  string    `json:"losscutPrice"`
				Status        string    `json:"status"`
				TimeInForce   string    `json:"timeInForce"`
				Timestamp     time.Time `json:"timestamp"`
			} `json:"list"`
		} `json:"data"`
		Responsetime time.Time `json:"responsetime"`
		Messages     []struct {
			MessageCode   string `json:"message_code"`
			MessageString string `json:"message_string"`
		} `json:"messages"`
	}
	resData := Res{}
	json.Unmarshal(res, &resData)
	if resData.Status != 0 {
		return nil, errors.New(fmt.Sprintf("%+v", resData.Messages))
	}

	// 返却値の作成
	ret := []order.Order{}
	for _, data := range resData.Data.List {
		//log.Printf("%+v\n", data)
		price, _ := strconv.ParseFloat(data.Price, 64)
		size, _ := strconv.ParseFloat(data.Size, 64)
		ret = append(ret, order.Order{
			ID: id.NewID(gmo.name, data.Symbol, fmt.Sprint(data.OrderID)),
			Request: order.Request{
				IsBuy:     data.Side == "BUY",
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

func (gmo *gmo) Stocks(symbol string) (stock.Stock, error) {
	type Req struct {
		Symbol string `json:"symbol"`
	}
	res, err := gmo.getRequest("/private/v1/openPositions", &Req{
		Symbol: symbol,
	})
	if err != nil {
		return stock.Stock{}, err
	}

	// レスポンスの変換
	type Res struct {
		Status int `json:"status"`
		Data   struct {
			Pagination struct {
				CurrentPage int `json:"currentPage"`
				Count       int `json:"count"`
			} `json:"pagination"`
			List []struct {
				PositionID   int       `json:"positionId"`
				Symbol       string    `json:"symbol"`
				Side         string    `json:"side"`
				Size         string    `json:"size"`
				OrderdSize   string    `json:"orderdSize"`
				Price        string    `json:"price"`
				LossGain     string    `json:"lossGain"`
				Leverage     string    `json:"leverage"`
				LosscutPrice string    `json:"losscutPrice"`
				Timestamp    time.Time `json:"timestamp"`
			} `json:"list"`
		} `json:"data"`
		Responsetime time.Time `json:"responsetime"`
		Messages     []struct {
			MessageCode   string `json:"message_code"`
			MessageString string `json:"message_string"`
		} `json:"messages"`
	}
	resData := Res{}
	json.Unmarshal(res, &resData)
	if resData.Status != 0 {
		return stock.Stock{}, errors.New(fmt.Sprintf("%+v", resData.Messages))
	}

	// 返却値の作成
	ret := stock.Stock{Symbol: symbol}
	for _, data := range resData.Data.List {
		size, _ := strconv.ParseFloat(data.Size, 64)
		if data.Side == "SELL" {
			ret.Size -= size
		} else {
			ret.Size += size
		}
	}

	return ret, nil
}

func (gmo *gmo) Balance() ([]base.Balance, error) {
	res, err := gmo.getRequest("/private/v1/account/assets", nil)
	if err != nil {
		return nil, err
	}

	// レスポンスの変換
	type Res struct {
		Status int `json:"status"`
		Data   []struct {
			Amount         string `json:"amount"`
			Available      string `json:"available"`
			ConversionRate string `json:"conversionRate"`
			Symbol         string `json:"symbol"`
		} `json:"data"`
		Responsetime time.Time `json:"responsetime"`
		Messages     []struct {
			MessageCode   string `json:"message_code"`
			MessageString string `json:"message_string"`
		} `json:"messages"`
	}
	resData := Res{}
	json.Unmarshal(res, &resData)
	if resData.Status != 0 {
		return []base.Balance{}, errors.New(fmt.Sprintf("%+v", resData.Messages))
	}

	// 返却値の作成
	ret := []base.Balance{}
	for _, data := range resData.Data {
		size, _ := strconv.ParseFloat(data.Amount, 64)
		ret = append(ret, base.Balance{
			CurrencyCode: data.Symbol,
			Size:         size,
		})
	}

	return ret, nil
}

func (gmo *gmo) Boards(symbol string) (board.Board, error) {
	type Req struct {
		Symbol string `json:"symbol"`
	}
	res, err := gmo.getRequest("/public/v1/orderbooks", &Req{
		Symbol: symbol,
	})
	if err != nil {
		return board.Board{}, err
	}

	// レスポンスの変換
	type Res struct {
		Status int `json:"status"`
		Data   struct {
			Asks []struct {
				Price string `json:"price"`
				Size  string `json:"size"`
			} `json:"asks"`
			Bids []struct {
				Price string `json:"price"`
				Size  string `json:"size"`
			} `json:"bids"`
			Symbol string `json:"symbol"`
		} `json:"data"`
		Responsetime time.Time `json:"responsetime"`
		Messages     []struct {
			MessageCode   string `json:"message_code"`
			MessageString string `json:"message_string"`
		} `json:"messages"`
	}
	resData := Res{}
	json.Unmarshal(res, &resData)
	if resData.Status != 0 {
		return board.Board{}, errors.New(fmt.Sprintf("%+v", resData.Messages))
	}

	// 返却値の作成
	asks := []base.Norm{}
	for _, v := range resData.Data.Asks {
		price, _ := strconv.ParseFloat(v.Price, 64)
		size, _ := strconv.ParseFloat(v.Size, 64)
		asks = append(asks, base.Norm{
			Price: price,
			Size:  size,
		})
	}
	bids := []base.Norm{}
	for _, v := range resData.Data.Bids {
		price, _ := strconv.ParseFloat(v.Price, 64)
		size, _ := strconv.ParseFloat(v.Size, 64)
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

func (gmo *gmo) InScheduledMaintenance() bool {
	// jst := utiltime.Jst()
	// // 355 <= time <= 415で落とす
	// from := 355
	// to := 415
	// now := jst.Hour()*100 + jst.Minute()
	// return from <= now && now <= to
	return false
}

func (gmo *gmo) getRequest(path string, param interface{}) ([]byte, error) {
	query := ""
	if param != nil {
		query = structToQuery(param)
	}

	url := url.URL{Scheme: "https", Host: gmo.host, Path: path, RawQuery: query}
	req, _ := http.NewRequest(
		"GET",
		url.String(),
		nil,
	)

	req.Header.Add("Content-Type", "application/json")

	pathSplited := strings.Split(path, "private")
	if len(pathSplited) > 1 {
		gmo.addSignature(req, "GET", pathSplited[1], nil)
	}

	return gmo.request(req)
}

func (gmo *gmo) postRequest(path string, param interface{}) ([]byte, error) {
	u := url.URL{Scheme: "https", Host: gmo.host, Path: path}
	jsonParam, _ := json.Marshal(param)
	req, _ := http.NewRequest(
		"POST",
		u.String(),
		bytes.NewBuffer(jsonParam),
	)

	req.Header.Add("Content-Type", "application/json")

	pathSplited := strings.Split(path, "private")
	if len(pathSplited) > 1 {
		gmo.addSignature(req, "POST", pathSplited[1], param)
	}

	return gmo.request(req)
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

func (gmo *gmo) addSignature(req *http.Request, method, path string, param interface{}) {
	key := gmo.key

	timestamp := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
	rawSig := timestamp + method + path
	if param != nil {
		byteParam, _ := json.Marshal(param)
		rawSig += string(byteParam)
	}

	hc := hmac.New(sha256.New, []byte(key.sec))
	hc.Write([]byte(rawSig))
	sign := hex.EncodeToString(hc.Sum(nil))

	req.Header.Add("API-KEY", key.id)
	req.Header.Add("API-TIMESTAMP", timestamp)
	req.Header.Add("API-SIGN", sign)
}

func (gmo *gmo) request(req *http.Request) ([]byte, error) {
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

func (gmo *gmo) UpdateLTP(lastTimePrice float64) error {
	return errors.New("not supported.")
}
