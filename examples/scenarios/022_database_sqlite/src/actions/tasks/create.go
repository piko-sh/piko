package tasks

import (
	"time"

	"piko.sh/piko"
	"piko.sh/piko/wdk/db"

	"testmodule/db/generated"
)

// CreateAction handles adding a new task to the database.
//
// The struct name determines how the action is called from the template:
//
//	action.tasks.Create($form)
//	       --+--  --+--
//	       package  struct name minus "Action" suffix
type CreateAction struct {
	piko.ActionMetadata
}

// CreateInput defines the data the action expects to receive.
//
// The json tags map HTML form field names to struct fields. When the form
// is submitted, Piko deserialises the form data using these tags.
type CreateInput struct {
	Title string `json:"title" validate:"required,min=1,max=500"`
}

// CreateResponse defines the data returned after a successful creation.
type CreateResponse struct {
	ID    int32  `json:"id"`
	Title string `json:"title"`
}

// Call inserts a new task into the database and returns its ID and title.
func (a CreateAction) Call(input CreateInput) (CreateResponse, error) {
	conn, err := db.GetDatabaseConnection("tasks")
	if err != nil {
		return CreateResponse{}, err
	}

	queries := generated.New(conn)
	row, err := queries.CreateTask(a.Ctx(), generated.CreateTaskParams{
		P1: input.Title,
		P2: int32(time.Now().Unix()),
	})
	if err != nil {
		return CreateResponse{}, err
	}

	return CreateResponse{
		ID:    row.ID,
		Title: row.Title,
	}, nil
}
