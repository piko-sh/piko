---
title: Testing what you built
description: Add pikotest coverage over the task manager. Component tests, action tests, and one snapshot.
nav:
  sidebar:
    section: "tutorials"
    subsection: "getting-started"
    order: 60
---

# Testing what you built

In this tutorial we will add tests to the task manager from [Data-backed pages with the querier](05-data-backed-pages.md). The goal is not full coverage but a working baseline. One component test for the task list, one action test for the Create action, one snapshot over the empty state. By the end the project has a test suite that catches regressions without needing a running server.

<p align="center">
  <img src="../diagrams/tutorial-06-preview.svg"
       alt="Preview of the finished test suite: a terminal window showing go test output. Three green passing lines cover a component test, an action test, and a snapshot test, with a final badge confirming three of three tests succeeded in under a second."
       width="500"/>
</p>

Piko ships two test substrates. Pikotest runs fast AST-level tests without a browser. The browser harness in `wdk/browser` drives a real Chromedp instance. This tutorial uses pikotest because it covers most of what a CRUD app needs. The browser harness sits alongside for later. See [how to browser testing](../how-to/browser-testing.md) for that path.

## Step 1: Pick the test layout

Place test files next to the generated component package that `piko generate` produces under `dist/`. A convenient convention is a sibling `_test.go` file:

```
dist/
  pages/
    pages_index_abc123/
      component.go         # generated
      component_test.go    # your test
  partials/
    partials_task_list_def456/
      component.go
      component_test.go
```

The test package imports the generated component package and the `piko.sh/piko` test helpers. Anything else (mocks, builders) lives in a package the test imports explicitly.

## Step 2: Set up a dependency-injected database for tests

Production code retrieves the database through `db.GetDatabaseConnection("tasks")`. Tests need a way to inject a different connection. The cleanest approach is context-based.

Create a helper at `testhelpers/db.go`:

```go
package testhelpers

import (
    "context"
    "database/sql"
    "testing"

    _ "modernc.org/sqlite"

    "myapp/db/generated"
)

// OpenMemoryDB returns a fresh in-memory SQLite connection seeded with the
// task schema. Each call gives a clean database.
func OpenMemoryDB(t *testing.T) *sql.DB {
    t.Helper()

    db, err := sql.Open("sqlite", ":memory:")
    if err != nil {
        t.Fatalf("open: %v", err)
    }
    t.Cleanup(func() { db.Close() })

    _, err = db.Exec(`
        CREATE TABLE tasks (
          id INTEGER PRIMARY KEY AUTOINCREMENT,
          title TEXT NOT NULL,
          completed INTEGER NOT NULL DEFAULT 0,
          created_at INTEGER NOT NULL
        );
    `)
    if err != nil {
        t.Fatalf("schema: %v", err)
    }

    return db
}

// SeedTasks inserts a small fixture for tests that need existing data.
func SeedTasks(t *testing.T, db *sql.DB, titles ...string) {
    t.Helper()
    q := generated.New(db)
    for i, title := range titles {
        if _, err := q.CreateTask(context.Background(), generated.CreateTaskParams{
            P1: title,
            P2: int32(1_700_000_000 + i),
        }); err != nil {
            t.Fatalf("seed: %v", err)
        }
    }
}
```

This helper gives every test a clean in-memory database. No shared state between tests. No database file to clean up. Ten tests run in under a second.

## Step 3: Make the app read the database from context

To let tests inject a connection, shift the production code to read the database through a context lookup with a bootstrap fallback. Create `dbconn/conn.go`:

```go
package dbconn

import (
    "context"
    "database/sql"

    "piko.sh/piko/wdk/db"
)

type ctxKey struct{}

// WithConn returns a context carrying the given database connection for
// testhelpers. Production callers do not use this.
func WithConn(ctx context.Context, c *sql.DB) context.Context {
    return context.WithValue(ctx, ctxKey{}, c)
}

// Conn returns the connection bound to the context, or the bootstrap-
// registered "tasks" connection if none is present.
func Conn(ctx context.Context) (*sql.DB, error) {
    if c, ok := ctx.Value(ctxKey{}).(*sql.DB); ok {
        return c, nil
    }
    return db.GetDatabaseConnection("tasks")
}
```

Update every action and partial to call `dbconn.Conn(a.Ctx())` or `dbconn.Conn(r.Context())` instead of `db.GetDatabaseConnection("tasks")`. Production behaviour is identical. Tests get a hook.

## Step 4: Write the first component test

Create `dist/partials/partials_task_list_xxx/component_test.go` (replace `xxx` with the hash Piko generated):

```go
package partials_task_list_xxx_test

import (
    "testing"

    "piko.sh/piko"

    taskList "myapp/dist/partials/partials_task_list_xxx"
    "myapp/dbconn"
    "myapp/testhelpers"
)

func TestTaskListPartial_EmptyState(t *testing.T) {
    db := testhelpers.OpenMemoryDB(t)
    ctx := dbconn.WithConn(context.Background(), db)

    req := piko.NewTestRequest("GET", "/").WithContext(ctx).Build()

    tester := piko.NewComponentTester(t, taskList.BuildAST)
    view := tester.Render(req, piko.NoProps{})

    view.QueryAST("ul.task-list").NotExists()
    view.QueryAST("p.empty-message").Exists()
    view.QueryAST("p.empty-message").ContainsText("No tasks yet")
}
```

The assertion pattern is AST-based. `QueryAST` accepts any CSS selector. The returned result chains assertions. No HTML rendering happens, which keeps the test fast and sidesteps the need for a configured render service.

Run the test:

```bash
go test ./dist/partials/partials_task_list_xxx/...
```

Expect a single pass.

## Step 5: Cover the populated state

Add a second test to the same file:

```go
func TestTaskListPartial_PopulatedState(t *testing.T) {
    db := testhelpers.OpenMemoryDB(t)
    testhelpers.SeedTasks(t, db, "buy milk", "walk the dog", "write tests")

    ctx := dbconn.WithConn(context.Background(), db)
    req := piko.NewTestRequest("GET", "/").WithContext(ctx).Build()

    tester := piko.NewComponentTester(t, taskList.BuildAST)
    view := tester.Render(req, piko.NoProps{})

    view.QueryAST("li.task-item").Count(3)
    view.QueryAST("li.task-item").First().ContainsText("write tests")
    view.QueryAST("p.empty-message").NotExists()
}
```

`SeedTasks` inserts three rows in a known order. The partial's `ORDER BY created_at DESC` puts "write tests" first because `SeedTasks` uses increasing timestamps.

Three assertions cover three things. The list has exactly three items. The first item matches the most-recent seed. The empty-state message is absent.

## Step 6: Write an action test

Create `actions/tasks/create_test.go`:

```go
package tasks_test

import (
    "context"
    "testing"

    "piko.sh/piko"

    "myapp/actions/tasks"
    "myapp/dbconn"
    "myapp/testhelpers"
)

func TestCreateAction_Success(t *testing.T) {
    db := testhelpers.OpenMemoryDB(t)
    ctx := dbconn.WithConn(context.Background(), db)

    tester := piko.NewActionTester(t, &tasks.CreateAction{})
    req := piko.NewTestRequest("POST", "/actions/tasks.Create").
        WithContext(ctx).
        WithFormData("title", "buy milk").
        Build()

    resp := tester.Invoke(req)

    tester.AssertSuccess(resp)
    if resp.Title != "buy milk" {
        t.Fatalf("got title %q, want buy milk", resp.Title)
    }
    if resp.ID == 0 {
        t.Fatal("expected a non-zero ID")
    }
}

func TestCreateAction_ValidationError(t *testing.T) {
    db := testhelpers.OpenMemoryDB(t)
    ctx := dbconn.WithConn(context.Background(), db)

    tester := piko.NewActionTester(t, &tasks.CreateAction{})
    req := piko.NewTestRequest("POST", "/actions/tasks.Create").
        WithContext(ctx).
        WithFormData("title", "").
        Build()

    resp := tester.Invoke(req)

    tester.AssertError(resp)
    tester.AssertHasFieldError(resp, "title")
}
```

Two tests cover the happy and sad paths. A successful create returns an ID and the title. An empty title trips the `validate:"required"` tag and the action returns a field-level error without ever talking to the database.

For the full action assertion surface see [testing API reference](../reference/testing-api.md#action-tester).

## Step 7: Snapshot the empty state

Snapshots guard against unintended structural or textual drift. One snapshot over a stable region catches a surprising amount.

Add to the partial test:

```go
func TestTaskListPartial_EmptyStateSnapshot(t *testing.T) {
    db := testhelpers.OpenMemoryDB(t)
    ctx := dbconn.WithConn(context.Background(), db)

    req := piko.NewTestRequest("GET", "/").WithContext(ctx).Build()
    tester := piko.NewComponentTester(t, taskList.BuildAST)
    view := tester.Render(req, piko.NoProps{})

    view.MatchSnapshot("task-list-empty")
}
```

The first run writes `__snapshots__/partials_task_list_xxx/task-list-empty.golden.html`. Later runs compare. A change to the empty-state HTML fails the test, which prompts the author to regenerate the snapshot with:

```bash
PIKO_UPDATE_SNAPSHOTS=1 go test ./dist/partials/partials_task_list_xxx/...
```

Snapshots belong in Git. A diff on `task-list-empty.golden.html` in a pull request makes visual changes reviewable.

## Step 8: Run everything

```bash
go test ./...
```

Every test runs in memory. No network, no browser, no external process. A laptop runs the whole suite in well under a second.

For coverage:

```bash
go test -cover ./...
```

For race-detection on concurrent code paths:

```bash
go test -race ./...
```

## What did not get tested

The tests above cover rendering, action success, and action validation. Three gaps remain:

- **Database-layer failures.** Simulating a disk-full error or a connection timeout needs a mock `*sql.DB`. Use `DATA-DOG/go-sqlmock` or a stub driver when this matters.
- **The dispatch pipeline.** Pikotest drives the action directly. CSRF, rate limiting, and other middleware do not run. [How to testing](../how-to/testing.md) and [browser testing reference](../reference/browser-testing.md) cover end-to-end flows.
- **The client-side glue.** The TypeScript in the partial (`handleToggle`, `handleDelete`) runs in the browser. Use the browser harness to cover it; see [how to browser testing](../how-to/browser-testing.md).

Resist the urge to test everything. The tests above give you confidence that the component renders correctly from the database, the Create action validates its input, and nothing regresses structurally. Add more only when a specific bug class keeps appearing.

## Next steps

- [Going multilingual](07-going-multilingual.md): add i18n to the blog from tutorial 04.
- [Testing API reference](../reference/testing-api.md) for the full builder, tester, and assertion surface.
- [How to testing](../how-to/testing.md) for mocking, benchmarking, and table-driven patterns.
- [How to browser testing](../how-to/browser-testing.md) when you need to cover client-side flows.
