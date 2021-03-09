package liquid

import (
	"bytes"
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
	"github.com/TTRSQ/ccew/domains/stock"
	"github.com/TTRSQ/ccew/interface/exchange"
	jwt "github.com/dgrijalva/jwt-go"
)

type liquid struct {
	apiKey    string
	apiSecKey string
	host      string
	name      string
}

var productIDMap map[string]int

func init() {
	productIDMap = map[string]int{
		"BTCJPY":     5,
		"ETHJPY":     29,
		"XRPJPY":     83,
		"BCHJPY":     41,
		"FX_BTCJPY":  5,
		"FX_ETHJPY":  29,
		"FX_XRPJPY":  83,
		"FX_BCHJPY":  41,
		"FX_QASHJPY": 50,

		// qash fiat
		"QASHJPY": 50,
		"QASHUSD": 57,
		"QASHEUR": 58,
		"QASHSGD": 59,
		"QASHHKD": 62,
		"QASHAUD": 60,
		"QASHPHP": 63, // disabled
		"QASHIDR": 61, // disabled
	}
}

// New return exchange obj.
func New(key exchange.Key) (exchange.Exchange, error) {
	lq := liquid{}
	lq.name = "liquid"
	lq.host = "api.liquid.com"

	if key.APIKey == "" || key.APISecKey == "" {
		return nil, errors.New("APIKey and APISecKey Required")
	}
	lq.apiKey = key.APIKey
	lq.apiSecKey = key.APISecKey

	return &lq, nil
}

func (lq *liquid) ExchangeName() string {
	return lq.name
}

func (lq *liquid) OrderTypes() exchange.OrderTypes {
	return exchange.OrderTypes{
		Limit:  "limit",
		Market: "market",
	}
}

func (lq *liquid) CreateOrder(price, size float64, isBuy bool, symbol, orderType string) (*order.ID, error) {
	// リクエスト
	type o struct {
		LeverageLevel interface{} `json:"leverage_level"`
		OrderType     string      `json:"order_type"`
		ProductID     int         `json:"product_id"`
		Side          string      `json:"side"`
		Quantity      float64     `json:"quantity"`
		Price         interface{} `json:"price"`
		MarginType    string      `json:"margin_type"`
	}
	type Req struct {
		Order o `json:"order"`
	}
	var leverageLevel interface{}
	if strings.HasPrefix(symbol, "FX_") {
		leverageLevel = 2
	}
	res, err := lq.postRequest("/orders", &Req{
		Order: o{
			ProductID:     productIDMap[symbol],
			OrderType:     orderType,
			Side:          map[bool]string{true: "buy", false: "sell"}[isBuy],
			Price:         map[bool]interface{}{true: price, false: nil}[orderType == lq.OrderTypes().Limit],
			Quantity:      size,
			LeverageLevel: leverageLevel,
			MarginType:    "isolated",
		},
	})
	if err != nil {
		return nil, err
	}

	// レスポンスの変換
	type Res struct {
		ID                   int         `json:"id"`
		OrderType            string      `json:"order_type"`
		MarginType           interface{} `json:"margin_type"`
		Quantity             string      `json:"quantity"`
		DiscQuantity         string      `json:"disc_quantity"`
		IcebergTotalQuantity string      `json:"iceberg_total_quantity"`
		Side                 string      `json:"side"`
		FilledQuantity       string      `json:"filled_quantity"`
		Price                string      `json:"price"`
		CreatedAt            int         `json:"created_at"`
		UpdatedAt            int         `json:"updated_at"`
		Status               string      `json:"status"`
		LeverageLevel        int         `json:"leverage_level"`
		SourceExchange       string      `json:"source_exchange"`
		ProductID            int         `json:"product_id"`
		ProductCode          string      `json:"product_code"`
		FundingCurrency      string      `json:"funding_currency"`
		CurrencyPairCode     string      `json:"currency_pair_code"`
		OrderFee             string      `json:"order_fee"`
		ClientOrderID        interface{} `json:"client_order_id"`
		ErrorMessage         string      `json:"message"`
		Errors               interface{} `json:"errors"`
	}
	resData := Res{}
	json.Unmarshal(res, &resData)
	if resData.ErrorMessage != "" {
		return nil, errors.New(resData.ErrorMessage)
	}

	return &order.ID{ExchangeName: lq.name, LocalID: fmt.Sprint(resData.ID)}, nil
}

func (lq *liquid) CancelOrder(symbol, localID string) error {

	_, err := lq.putRequest("/orders/"+localID+"/cancel", nil)
	return err
}

func (lq *liquid) CancelAllOrder(symbol string) error {

	return errors.New("not supported.")
}

func (lq *liquid) ActiveOrders(symbol string) ([]order.Order, error) {
	type Req struct {
		Symbol     int    `json:"product_id"`
		OrderState string `json:"status"`
	}
	res, err := lq.getRequest("/orders", &Req{
		OrderState: "live",
		Symbol:     productIDMap[symbol],
	})
	if err != nil {
		return nil, err
	}

	// レスポンスの変換
	type Res struct {
		Models []struct {
			ID                   int           `json:"id"`
			OrderType            string        `json:"order_type"`
			MarginType           interface{}   `json:"margin_type"`
			Quantity             string        `json:"quantity"`
			DiscQuantity         string        `json:"disc_quantity"`
			IcebergTotalQuantity string        `json:"iceberg_total_quantity"`
			Side                 string        `json:"side"`
			FilledQuantity       string        `json:"filled_quantity"`
			Price                string        `json:"price"`
			CreatedAt            int           `json:"created_at"`
			UpdatedAt            int           `json:"updated_at"`
			Status               string        `json:"status"`
			LeverageLevel        int           `json:"leverage_level"`
			SourceExchange       string        `json:"source_exchange"`
			ProductID            int           `json:"product_id"`
			ProductCode          string        `json:"product_code"`
			FundingCurrency      string        `json:"funding_currency"`
			CurrencyPairCode     string        `json:"currency_pair_code"`
			OrderFee             string        `json:"order_fee"`
			Executions           []interface{} `json:"executions"`
		} `json:"models"`
		CurrentPage int `json:"current_page"`
		TotalPages  int `json:"total_pages"`
	}
	resData := Res{}
	json.Unmarshal(res, &resData)

	// 返却値の作成
	ret := []order.Order{}
	for _, data := range resData.Models {
		//log.Printf("%+v\n", data)
		price, _ := strconv.ParseFloat(data.Quantity, 64)
		size, _ := strconv.ParseFloat(data.Quantity, 64)
		ret = append(ret, order.Order{
			ID: order.NewID(lq.name, data.CurrencyPairCode, fmt.Sprint(data.ID)),
			Request: order.Request{
				IsBuy:     data.Side == "buy",
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

func (lq *liquid) Stocks(symbol string) (stock.Stock, error) {
	type Req struct {
		Symbol int    `json:"product_id"`
		Status string `json:"status"`
	}
	res, err := lq.getRequest("/trades", &Req{
		Symbol: productIDMap[symbol],
		Status: "open",
	})
	if err != nil {
		return stock.Stock{}, err
	}

	// レスポンスの変換
	type Res struct {
		Models []struct {
			ID                int         `json:"id"`
			CurrencyPairCode  string      `json:"currency_pair_code"`
			Status            string      `json:"status"`
			Side              string      `json:"side"`
			MarginType        string      `json:"margin_type"`
			MarginUsed        string      `json:"margin_used"`
			LiquidationPrice  interface{} `json:"liquidation_price"`
			MaintenanceMargin interface{} `json:"maintenance_margin"`
			OpenQuantity      string      `json:"open_quantity"`
			CloseQuantity     string      `json:"close_quantity"`
			Quantity          string      `json:"quantity"`
			LeverageLevel     int         `json:"leverage_level"`
			ProductCode       string      `json:"product_code"`
			ProductID         int         `json:"product_id"`
			OpenPrice         string      `json:"open_price"`
			ClosePrice        string      `json:"close_price"`
			TraderID          int         `json:"trader_id"`
			OpenPnl           string      `json:"open_pnl"`
			ClosePnl          string      `json:"close_pnl"`
			Pnl               string      `json:"pnl"`
			StopLoss          string      `json:"stop_loss"`
			TakeProfit        string      `json:"take_profit"`
			FundingCurrency   string      `json:"funding_currency"`
			CreatedAt         int         `json:"created_at"`
			UpdatedAt         int         `json:"updated_at"`
			TotalInterest     string      `json:"total_interest"`
		} `json:"models"`
		CurrentPage int `json:"current_page"`
		TotalPages  int `json:"total_pages"`
	}
	resData := Res{}
	json.Unmarshal(res, &resData)

	// 返却値の作成
	ret := stock.Stock{Symbol: symbol}
	for _, data := range resData.Models {
		size, _ := strconv.ParseFloat(data.Quantity, 64)
		if data.Side == "sell" {
			ret.Size -= size
		} else {
			ret.Size += size
		}
	}

	return ret, nil
}

func (lq *liquid) Balance() ([]base.Balance, error) {
	res, err := lq.getRequest("/accounts/balance", nil)
	if err != nil {
		return nil, err
	}

	// レスポンスの変換
	type Res []struct {
		Currency string `json:"currency"`
		Balance  string `json:"balance"`
	}
	resData := Res{}
	json.Unmarshal(res, &resData)

	// 返却値の作成
	ret := []base.Balance{}
	for _, data := range resData {
		size, _ := strconv.ParseFloat(data.Balance, 64)
		ret = append(ret, base.Balance{
			CurrencyCode: data.Currency,
			Size:         size,
		})
	}

	return ret, nil
}

func (lq *liquid) Boards(symbol string) (board.Board, error) {
	res, err := lq.getRequest("/products/"+fmt.Sprint(productIDMap[symbol])+"/price_levels", nil)
	if err != nil {
		return board.Board{}, err
	}

	// レスポンスの変換
	type Res struct {
		Bids      [][]string `json:"buy_price_levels"`
		Asks      [][]string `json:"sell_price_levels"`
		Timestamp string     `json:"timestamp"`
	}
	resData := Res{}
	json.Unmarshal(res, &resData)

	// 返却値の作成
	asks := []base.Norm{}
	for _, v := range resData.Asks {
		price, _ := strconv.ParseFloat(v[0], 64)
		size, _ := strconv.ParseFloat(v[1], 64)
		asks = append(asks, base.Norm{
			Price: price,
			Size:  size,
		})
	}
	bids := []base.Norm{}
	for _, v := range resData.Bids {
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

func (lq *liquid) InScheduledMaintenance() bool {
	// jst := utiltime.Jst()
	// // 355 <= time <= 415で落とす
	// from := 355
	// to := 415
	// now := jst.Hour()*100 + jst.Minute()
	// return from <= now && now <= to
	return false
}

func (lq *liquid) getRequest(path string, param interface{}) ([]byte, error) {
	jsonParam, _ := json.Marshal(param)

	query := ""
	if param != nil {
		query = structToQuery(param)
	}

	url := url.URL{Scheme: "https", Host: lq.host, Path: path, RawQuery: query}
	req, _ := http.NewRequest(
		"GET",
		url.String(),
		bytes.NewBuffer(jsonParam),
	)

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Quoine-Auth", lq.makeSignature(path+"?"+query))
	req.Header.Add("X-Quoine-API-Version", "2")

	return lq.request(req)
}

func (lq *liquid) putRequest(path string, param interface{}) ([]byte, error) {
	jsonParam, _ := json.Marshal(param)

	query := ""
	if param != nil {
		query = structToQuery(param)
	}

	url := url.URL{Scheme: "https", Host: lq.host, Path: path, RawQuery: query}
	req, _ := http.NewRequest(
		"PUT",
		url.String(),
		bytes.NewBuffer(jsonParam),
	)

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Quoine-Auth", lq.makeSignature(path+"?"+query))
	req.Header.Add("X-Quoine-API-Version", "2")

	return lq.request(req)
}

func (lq *liquid) postRequest(path string, param interface{}) ([]byte, error) {
	u := url.URL{Scheme: "https", Host: lq.host, Path: path}
	jsonParam, _ := json.Marshal(param)
	req, _ := http.NewRequest(
		"POST",
		u.String(),
		bytes.NewBuffer(jsonParam),
	)

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Quoine-Auth", lq.makeSignature(path))
	req.Header.Add("X-Quoine-API-Version", "2")

	return lq.request(req)
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

func (lq *liquid) makeSignature(path string) string {

	mySigningKey := []byte(lq.apiSecKey)

	type MyCustomClaims struct {
		Path    string `json:"path"`
		Nonce   string `json:"nonce"`
		TokenId string `json:"token_id"`
		jwt.StandardClaims
	}

	// Create the Claims
	nonce := time.Now().UnixNano() / 1000000

	claims := MyCustomClaims{
		path,
		fmt.Sprintf("%d", nonce),
		fmt.Sprintf("%s", lq.apiKey),
		jwt.StandardClaims{},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	sig, _ := token.SignedString(mySigningKey)

	return sig
}

func (lq *liquid) request(req *http.Request) ([]byte, error) {
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
