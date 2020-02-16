package base

// Norm norm of something (e.g. Order, Position, Stock)
type Norm struct {
	Price float64
	Size  float64
}

// Currency information for identify currency
type Currency struct {
	ExchangeName string
	Symbol       string
}
