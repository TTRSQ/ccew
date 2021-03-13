package ftx

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

type ftx struct {
	name string
	host string
	key  exchange.Key
}

// New return exchange obj.
func New(key exchange.Key) (exchange.Exchange, error) {
	ftx := ftx{}
	ftx.name = "ftx"
	ftx.host = "ftx.com"

	if key.APIKey == "" || key.APISecKey == "" {
		return nil, errors.New("APIKey and APISecKey Required")
	}
	ftx.key = key

	return &ftx, nil
}

func (ftx *ftx) ExchangeName() string {
	return ftx.name
}

func (ftx *ftx) OrderTypes() exchange.OrderTypes {
	return exchange.OrderTypes{
		Limit:  "limit",
		Market: "market",
	}
}

func (ftx *ftx) CreateOrder(price, size float64, isBuy bool, symbol, orderType string) (*order.Responce, error) {
	// リクエスト
	type Req struct {
		Market string   `json:"market"`
		Type   string   `json:"type"`
		Side   string   `json:"side"`
		Price  *float64 `json:"price"`
		Size   float64  `json:"size"`
	}

	res, err := ftx.postRequest("/api/orders", Req{
		Market: symbol,
		Type:   orderType,
		Side:   map[bool]string{true: "buy", false: "sell"}[isBuy],
		Price:  map[bool]*float64{true: &price, false: nil}[orderType == ftx.OrderTypes().Limit],
		Size:   size,
	})

	if err != nil {
		return nil, err
	}

	// レスポンスの変換

	type Res struct {
		Success bool `json:"success"`
		Result  struct {
			CreatedAt     time.Time   `json:"createdAt"`
			FilledSize    float64     `json:"filledSize"`
			Future        string      `json:"future"`
			ID            int         `json:"id"`
			Market        string      `json:"market"`
			Price         float64     `json:"price"`
			RemainingSize int         `json:"remainingSize"`
			Side          string      `json:"side"`
			Size          int         `json:"size"`
			Status        string      `json:"status"`
			Type          string      `json:"type"`
			ReduceOnly    bool        `json:"reduceOnly"`
			Ioc           bool        `json:"ioc"`
			PostOnly      bool        `json:"postOnly"`
			ClientID      interface{} `json:"clientId"`
		} `json:"result"`
	}
	resData := Res{}
	json.Unmarshal(res, &resData)

	return &order.Responce{
		ID:         id.NewID(ftx.name, symbol, fmt.Sprint(resData.Result.ID)),
		FilledSize: resData.Result.FilledSize,
	}, nil
}

func (ftx *ftx) CancelOrder(symbol, localID string) error {
	_, err := ftx.deleteRequest("/api/orders/"+localID, nil)
	return err
}

func (ftx *ftx) CancelAllOrder(symbol string) error {
	type Req struct {
		Market string `json:"market"`
	}

	_, err := ftx.deleteRequest("/api/orders", Req{
		Market: symbol,
	})
	return err
}

func (ftx *ftx) ActiveOrders(symbol string) ([]order.Order, error) {
	type Req struct {
		Market string `json:"market"`
	}
	res, _ := ftx.getRequest("/api/orders", Req{
		Market: symbol,
	})

	// レスポンスの変換

	type Res struct {
		Success bool `json:"success"`
		Result  []struct {
			CreatedAt     time.Time   `json:"createdAt"`
			FilledSize    int         `json:"filledSize"`
			Future        string      `json:"future"`
			ID            int         `json:"id"`
			Market        string      `json:"market"`
			Price         float64     `json:"price"`
			AvgFillPrice  float64     `json:"avgFillPrice"`
			RemainingSize float64     `json:"remainingSize"`
			Side          string      `json:"side"`
			Size          int         `json:"size"`
			Status        string      `json:"status"`
			Type          string      `json:"type"`
			ReduceOnly    bool        `json:"reduceOnly"`
			Ioc           bool        `json:"ioc"`
			PostOnly      bool        `json:"postOnly"`
			ClientID      interface{} `json:"clientId"`
		} `json:"result"`
	}

	resData := Res{}
	json.Unmarshal(res, &resData)

	// 返却値の作成
	ret := []order.Order{}
	for _, data := range resData.Result {
		//log.Printf("%+v\n", data)
		ret = append(ret, order.Order{
			ID: id.NewID(ftx.name, data.Market, fmt.Sprint(data.ID)),
			Request: order.Request{
				IsBuy:     data.Side == "buy",
				OrderType: data.Type,
				Norm: base.Norm{
					Price: float64(data.Price),
					Size:  data.RemainingSize,
				},
			},
		})
	}
	return ret, nil
}

func (ftx *ftx) Stocks(symbol string) (stock.Stock, error) {
	type Req struct {
		Symbol string `json:"product_code"`
	}
	res, _ := ftx.getRequest("/api/positions", nil)

	// レスポンスの変換
	type Res struct {
		Success bool `json:"success"`
		Result  []struct {
			Cost                         float64 `json:"cost"`
			EntryPrice                   float64 `json:"entryPrice"`
			EstimatedLiquidationPrice    float64 `json:"estimatedLiquidationPrice"`
			Future                       string  `json:"future"`
			InitialMarginRequirement     float64 `json:"initialMarginRequirement"`
			LongOrderSize                float64 `json:"longOrderSize"`
			MaintenanceMarginRequirement float64 `json:"maintenanceMarginRequirement"`
			NetSize                      float64 `json:"netSize"`
			OpenSize                     float64 `json:"openSize"`
			RealizedPnl                  float64 `json:"realizedPnl"`
			ShortOrderSize               float64 `json:"shortOrderSize"`
			Side                         string  `json:"side"`
			Size                         float64 `json:"size"`
			UnrealizedPnl                int     `json:"unrealizedPnl"`
			CollateralUsed               float64 `json:"collateralUsed"`
		} `json:"result"`
	}
	resData := Res{}
	json.Unmarshal(res, &resData)

	// 返却値の作成
	posMap := map[string]float64{}
	for _, data := range resData.Result {
		posMap[data.Future] = data.Size
	}

	size := 0.0
	val, exist := posMap[symbol]
	if exist {
		size = val
	}

	return stock.Stock{Symbol: symbol, Size: size}, nil
}

func (ftx *ftx) Balance() ([]base.Balance, error) {

	res, _ := ftx.getRequest("/api/wallet/balances", nil)

	// レスポンスの変換
	type Res struct {
		Success bool `json:"success"`
		Result  []struct {
			Coin  string  `json:"coin"`
			Free  float64 `json:"free"`
			Total float64 `json:"total"`
		} `json:"result"`
	}
	resData := Res{}
	json.Unmarshal(res, &resData)

	// 返却値の作成
	ret := []base.Balance{}
	for _, data := range resData.Result {
		ret = append(ret, base.Balance{
			CurrencyCode: data.Coin,
			Size:         data.Total,
		})
	}

	return ret, nil
}

func (ftx *ftx) Boards(symbol string) (board.Board, error) {
	type Req struct {
		Depth int `json:"depth"`
	}
	// /markets/{market_name}/orderbook?depth={depth}
	res, _ := ftx.getRequest("/api/markets/"+symbol+"/orderbook", Req{
		Depth: 50,
	})

	// レスポンスの変換
	type Res struct {
		Success bool `json:"success"`
		Result  struct {
			Asks [][]float64 `json:"asks"`
			Bids [][]float64 `json:"bids"`
		} `json:"result"`
	}

	resData := Res{}
	json.Unmarshal(res, &resData)

	// 返却値の作成
	asks := []base.Norm{}
	for _, v := range resData.Result.Asks {
		asks = append(asks, base.Norm{
			Price: v[0],
			Size:  v[1],
		})
	}
	bids := []base.Norm{}
	for _, v := range resData.Result.Bids {
		bids = append(bids, base.Norm{
			Price: v[0],
			Size:  v[1],
		})
	}

	bestAsk := resData.Result.Asks[0][0]
	bestBid := resData.Result.Bids[0][0]

	return board.Board{
		Symbol:   symbol,
		MidPrice: (bestAsk + bestBid) / 2.0,
		Asks:     asks,
		Bids:     bids,
	}, nil
}

func (ftx *ftx) InScheduledMaintenance() bool {
	// jst := utiltime.Jst()
	// // 355 <= time <= 415で落とす
	// from := 355
	// to := 415
	// now := jst.Hour()*100 + jst.Minute()
	// return from <= now && now <= to
	return false
}

func (ftx *ftx) getRequest(path string, param interface{}) ([]byte, error) {
	uparm := url.Values{}

	if param != nil {
		// jsonをガリガリする
		jsonParam, _ := json.Marshal(param)
		jsonStr := strings.Trim(string(jsonParam), "{}")
		jstrs := strings.Split(jsonStr, ",")
		for _, v := range jstrs {
			vv := strings.Split(v, ":")
			uparm.Add(strings.Trim(vv[0], "\""), strings.Trim(vv[1], "\""))
		}
	}

	uri := url.URL{Scheme: "https", Host: ftx.host, Path: path, RawQuery: uparm.Encode()}
	timestamp := fmt.Sprintf("%d", time.Now().UnixNano()/1000000)
	req, err := http.NewRequest(
		"GET",
		uri.String(),
		nil,
	)

	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("FTX-KEY", ftx.key.APIKey)
	req.Header.Add("FTX-TS", timestamp)

	rawSign := timestamp + "GET" + path
	if uparm.Encode() != "" {
		rawSign += "?" + uparm.Encode()
	}
	sign := ftx.makeHMAC(rawSign)

	req.Header.Add("FTX-SIGN", sign)
	val, exest := ftx.key.SpecificParam["FTX-SUBACCOUNT"]
	if exest {
		req.Header.Add("FTX-SUBACCOUNT", fmt.Sprint(val))
	}

	return ftx.request(req)
}

func (ftx *ftx) postRequest(path string, param interface{}) ([]byte, error) {
	url := url.URL{Scheme: "https", Host: ftx.host, Path: path}
	timestamp := fmt.Sprintf("%d", time.Now().UnixNano()/1000000)
	jsonParam, _ := json.Marshal(param)
	req, _ := http.NewRequest(
		"POST",
		url.String(),
		bytes.NewBuffer(jsonParam),
	)

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("FTX-KEY", ftx.key.APIKey)
	req.Header.Add("FTX-TS", timestamp)

	rawSign := timestamp + "POST" + path + string(jsonParam)
	sign := ftx.makeHMAC(rawSign)

	req.Header.Add("FTX-SIGN", sign)
	val, exest := ftx.key.SpecificParam["FTX-SUBACCOUNT"]
	if exest {
		req.Header.Add("FTX-SUBACCOUNT", fmt.Sprint(val))
	}

	return ftx.request(req)
}

func (ftx *ftx) deleteRequest(path string, param interface{}) ([]byte, error) {
	url := url.URL{Scheme: "https", Host: ftx.host, Path: path}
	timestamp := fmt.Sprintf("%d", time.Now().UnixNano()/1000000)
	jsonParam, _ := json.Marshal(param)
	req, _ := http.NewRequest(
		"DELETE",
		url.String(),
		bytes.NewBuffer(jsonParam),
	)

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("FTX-KEY", ftx.key.APIKey)
	req.Header.Add("FTX-TS", timestamp)

	rawSign := timestamp + "DELETE" + path + string(jsonParam)
	sign := ftx.makeHMAC(rawSign)

	req.Header.Add("FTX-SIGN", sign)
	val, exest := ftx.key.SpecificParam["FTX-SUBACCOUNT"]
	if exest {
		req.Header.Add("FTX-SUBACCOUNT", fmt.Sprint(val))
	}

	return ftx.request(req)
}

func (ftx *ftx) makeHMAC(msg string) string {
	mac := hmac.New(sha256.New, []byte(ftx.key.APISecKey))
	mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil))
}

func (ftx *ftx) request(req *http.Request) ([]byte, error) {
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
	if err != nil {
		log.Fatal(err)
	}

	type errCheck struct {
		Success bool `json:"success"`
	}
	check := errCheck{}
	err = json.Unmarshal(body, &check)
	if !check.Success || err != nil {
		msg := "request not accepted."
		if err != nil {
			msg = err.Error()
		}
		return nil, errors.New(msg)
	}

	return body, err
}

func (fx *ftx) UpdateLTP(lastTimePrice float64) error {
	return errors.New("not supported.")
}
