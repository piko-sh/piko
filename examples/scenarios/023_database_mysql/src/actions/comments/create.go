package comments

import (
	"time"

	"piko.sh/piko"
	"piko.sh/piko/wdk/db"

	"testmodule/db/generated"
)

// CreateAction handles adding a new comment to a blog post.
//
// The struct name determines how the action is called from the template:
//
//	action.comments.Create($form)
//	       ---+---  --+--
//	       package   struct name minus "Action" suffix
type CreateAction struct {
	piko.ActionMetadata
}

// CreateInput defines the data the action expects to receive.
//
// The json tags map HTML form field names to struct fields. When the form
// is submitted, Piko deserialises the form data using these tags.
type CreateInput struct {
	PostID     int32  `json:"post_id" validate:"required"`
	AuthorName string `json:"author_name" validate:"required,min=1,max=255"`
	Body       string `json:"body" validate:"required,min=1"`
}

// CreateResponse is returned after a successful creation. It is intentionally
// empty because the client reloads the page to reflect the change.
type CreateResponse struct{}

// Call inserts a new comment for the specified post.
func (a CreateAction) Call(input CreateInput) (CreateResponse, error) {
	conn, err := db.GetDatabaseConnection("blog")
	if err != nil {
		return CreateResponse{}, err
	}

	queries := generated.New(conn)
	if err := queries.CreateComment(a.Ctx(), generated.CreateCommentParams{
		P1: input.PostID,
		P2: input.AuthorName,
		P3: input.Body,
		P4: int32(time.Now().Unix()),
	}); err != nil {
		return CreateResponse{}, err
	}

	return CreateResponse{}, nil
}
