package tasks

import (
	"piko.sh/piko"
	"piko.sh/piko/wdk/db"

	"testmodule/db/generated"
)

// DeleteAction handles removing a task from the database.
//
// The struct name determines how the action is called from the template:
//
//	action.tasks.Delete({ id: 1 })
//	       --+--  --+---
//	       package  struct name minus "Action" suffix
type DeleteAction struct {
	piko.ActionMetadata
}

// DeleteInput defines the data the action expects to receive.
type DeleteInput struct {
	ID int32 `json:"id" validate:"required"`
}

// DeleteResponse is returned after a successful deletion. It is intentionally
// empty because the client reloads the page to reflect the change.
type DeleteResponse struct{}

// Call removes the specified task from the database.
func (a DeleteAction) Call(input DeleteInput) (DeleteResponse, error) {
	conn, err := db.GetDatabaseConnection("tasks")
	if err != nil {
		return DeleteResponse{}, err
	}

	queries := generated.New(conn)
	if err := queries.DeleteTask(a.Ctx(), input.ID); err != nil {
		return DeleteResponse{}, err
	}

	return DeleteResponse{}, nil
}
