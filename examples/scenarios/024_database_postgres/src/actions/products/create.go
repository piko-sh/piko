package products

import (
	"time"

	"piko.sh/piko"
	"piko.sh/piko/wdk/db"

	"testmodule/db/generated"
)

// CreateAction handles adding a new product to the database.
//
// The struct name determines how the action is called from the template:
//
//	action.products.Create($form)
//	       --------  ------
//	       package   struct name minus "Action" suffix
type CreateAction struct {
	piko.ActionMetadata
}

// CreateInput defines the data the action expects to receive.
//
// The json tags map HTML form field names to struct fields. When the form
// is submitted, Piko deserialises the form data using these tags.
type CreateInput struct {
	Name     string  `json:"name" validate:"required,min=1,max=255"`
	Category string  `json:"category" validate:"required,min=1,max=100"`
	Price    float64 `json:"price" validate:"required,gt=0"`
}

// CreateResponse defines the data returned after a successful creation.
type CreateResponse struct {
	ID   int32  `json:"id"`
	Name string `json:"name"`
}

// Call inserts a new product into the database and returns its ID and name.
func (a CreateAction) Call(input CreateInput) (CreateResponse, error) {
	conn, err := db.GetDatabaseConnection("analytics")
	if err != nil {
		return CreateResponse{}, err
	}

	queries := generated.New(conn)
	row, err := queries.CreateProduct(a.Ctx(), generated.CreateProductParams{
		P1: input.Name,
		P2: input.Category,
		P3: input.Price,
		P4: time.Now().Unix(),
	})
	if err != nil {
		return CreateResponse{}, err
	}

	return CreateResponse{
		ID:   row.ID,
		Name: row.Name,
	}, nil
}
