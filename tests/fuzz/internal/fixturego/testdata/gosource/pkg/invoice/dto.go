package invoice

type Order struct {
	Customer, SKU, Line   string
	Quantity, Unit, Total int
}
