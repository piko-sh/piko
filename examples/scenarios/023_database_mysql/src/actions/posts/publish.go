package posts

import (
	"piko.sh/piko"
	"piko.sh/piko/wdk/db"

	"testmodule/db/generated"
)

// PublishAction handles publishing a draft blog post.
//
// The struct name determines how the action is called from the template:
//
//	action.posts.Publish({ id: 1 })
//	       --+--  --+---
//	       package  struct name minus "Action" suffix
type PublishAction struct {
	piko.ActionMetadata
}

// PublishInput defines the data the action expects to receive.
type PublishInput struct {
	ID int32 `json:"id" validate:"required"`
}

// PublishResponse is returned after a successful publish. It is intentionally
// empty because the client reloads the page to reflect the change.
type PublishResponse struct{}

// Call marks the specified post as published.
func (a PublishAction) Call(input PublishInput) (PublishResponse, error) {
	conn, err := db.GetDatabaseConnection("blog")
	if err != nil {
		return PublishResponse{}, err
	}

	queries := generated.New(conn)
	if err := queries.PublishPost(a.Ctx(), input.ID); err != nil {
		return PublishResponse{}, err
	}

	return PublishResponse{}, nil
}
