package tasks

import (
	"piko.sh/piko"
	"piko.sh/piko/wdk/db"

	"testmodule/db/generated"
)

// ToggleAction handles toggling a task's completed state.
//
// The struct name determines how the action is called from the template:
//
//	action.tasks.Toggle({ id: 1 })
//	       --+--  --+---
//	       package  struct name minus "Action" suffix
type ToggleAction struct {
	piko.ActionMetadata
}

// ToggleInput defines the data the action expects to receive.
type ToggleInput struct {
	ID int32 `json:"id" validate:"required"`
}

// ToggleResponse is returned after a successful toggle. It is intentionally
// empty because the client reloads the page to reflect the change.
type ToggleResponse struct{}

// Call flips the completed flag on the specified task.
func (a ToggleAction) Call(input ToggleInput) (ToggleResponse, error) {
	conn, err := db.GetDatabaseConnection("tasks")
	if err != nil {
		return ToggleResponse{}, err
	}

	queries := generated.New(conn)
	if err := queries.ToggleComplete(a.Ctx(), input.ID); err != nil {
		return ToggleResponse{}, err
	}

	return ToggleResponse{}, nil
}
