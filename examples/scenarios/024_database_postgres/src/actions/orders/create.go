package orders

import (
	"fmt"
	"time"

	"piko.sh/piko"
	"piko.sh/piko/wdk/db"

	"testmodule/db/generated"
)

// CreateAction handles placing a new order in the database.
//
// The struct name determines how the action is called from the template:
//
//	action.orders.Create($form)
//	       ------  ------
//	       package  struct name minus "Action" suffix
type CreateAction struct {
	piko.ActionMetadata
}

// CreateInput defines the data the action expects to receive.
//
// The json tags map HTML form field names to struct fields. When the form
// is submitted, Piko deserialises the form data using these tags.
type CreateInput struct {
	ProductID int32 `json:"product_id" validate:"required,gt=0"`
	Quantity  int32 `json:"quantity" validate:"required,gt=0"`
}

// CreateResponse defines the data returned after a successful order creation.
type CreateResponse struct {
	ID    int32   `json:"id"`
	Total float64 `json:"total"`
}

// Call looks up the product price, calculates the total from price x quantity,
// and inserts a new order into the database.
func (a CreateAction) Call(input CreateInput) (CreateResponse, error) {
	conn, err := db.GetDatabaseConnection("analytics")
	if err != nil {
		return CreateResponse{}, err
	}

	queries := generated.New(conn)

	// Fetch the product to get its price.
	product, err := queries.GetProduct(a.Ctx(), input.ProductID)
	if err != nil {
		return CreateResponse{}, fmt.Errorf("fetching product %d: %w", input.ProductID, err)
	}

	total := product.Price * float64(input.Quantity)

	order, err := queries.CreateOrder(a.Ctx(), generated.CreateOrderParams{
		P1: input.ProductID,
		P2: input.Quantity,
		P3: total,
		P4: time.Now().Unix(),
	})
	if err != nil {
		return CreateResponse{}, err
	}

	return CreateResponse{
		ID:    order.ID,
		Total: order.Total,
	}, nil
}
