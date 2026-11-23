package workflowdata

import (
	"context"
	"strconv"

	"example.org/pikofuzzcase/db/generated"
	"piko.sh/piko"
	"piko.sh/piko/wdk/db"
)

type Task struct {
	ID     int64  `json:"id"`
	Label  string `json:"label"`
	Done   bool   `json:"done"`
	Marker string `json:"-"`
}

func task(id int64, label string, done bool) Task {
	return Task{ID: id, Label: label, Done: done, Marker: strconv.FormatBool(done)}
}

func queries() (*generated.Queries, error) {
	conn, err := db.GetDatabaseConnection("tasks")
	if err != nil {
		return nil, err
	}
	return generated.New(conn), nil
}

func List(ctx context.Context) ([]Task, error) {
	q, err := queries()
	if err != nil {
		return nil, err
	}
	rows, err := q.ListTasks(ctx)
	if err != nil {
		return nil, err
	}
	tasks := make([]Task, 0, len(rows))
	for _, row := range rows {
		tasks = append(tasks, task(row.ID, row.Label, row.Done != 0))
	}
	return tasks, nil
}

func Create(ctx context.Context, label string) (Task, error) {
	if label == "" {
		return Task{}, piko.NewValidationError(map[string]string{"label": "Label is required"})
	}
	q, err := queries()
	if err != nil {
		return Task{}, err
	}
	row, err := q.CreateTask(ctx, label)
	return task(row.ID, row.Label, row.Done != 0), err
}

func Toggle(ctx context.Context, id int64) (Task, error) {
	conn, err := db.GetDatabaseConnection("tasks")
	if err != nil {
		return Task{}, err
	}
	q := generated.New(conn)
	var changed Task
	err = q.RunInTx(ctx, conn, func(tx *generated.Queries) error {
		if err := tx.ToggleTask(ctx, id); err != nil {
			return err
		}
		row, err := tx.GetTask(ctx, id)
		changed = task(row.ID, row.Label, row.Done != 0)
		return err
	})
	return changed, err
}

func Delete(ctx context.Context, id int64) (Task, error) {
	conn, err := db.GetDatabaseConnection("tasks")
	if err != nil {
		return Task{}, err
	}
	q := generated.New(conn)
	var previous Task
	err = q.RunInTx(ctx, conn, func(tx *generated.Queries) error {
		row, err := tx.GetTask(ctx, id)
		if err != nil {
			return err
		}
		previous = task(row.ID, row.Label, row.Done != 0)
		return tx.DeleteTask(ctx, id)
	})
	return previous, err
}
