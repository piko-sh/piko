// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package generate

import (
	_ "embed"
	"fmt"
	"slices"
	"strings"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/fixturego"
)

const (
	// formVariantExtraRow selects the history that creates, toggles and deletes an extra
	// row.
	formVariantExtraRow = 1

	// formVariantDuplicate selects the history that repeats a step with duplicate labels.
	formVariantDuplicate = 2

	// formVariantReset selects the history that resets and resubmits the form.
	formVariantReset = 3

	// formExtraRowID is the identifier of the row created after the base history.
	formExtraRowID = 3

	// formDuplicateAt is the history position at which the duplicated step is inserted.
	formDuplicateAt = 6

	// formResetAt is the history position at which the reset commands are inserted.
	formResetAt = 7
)

var (
	// formWorkflowStore is the formatted Go source staged in the generated project.
	formWorkflowStore = fixturego.Fixed("pkg/workflowdata/data.go")

	// formWorkflowPage embeds the workflow page source.
	//
	//go:embed form_workflow_page.pk.tmpl
	formWorkflowPage string

	// formWorkflowList embeds the workflow list partial.
	//
	//go:embed form_workflow_list.pk.tmpl
	formWorkflowList string

	// formWorkflowField embeds the workflow field component.
	//
	//go:embed form_workflow_field.pkc.tmpl
	formWorkflowField string
)

// formHistory models a finite form-command history.
//
// Takes first (string) which is the first generated row label.
// Takes second (string) which is the second generated row label.
// Takes toggleSecond (bool) which selects whether the second row is toggled.
// Takes variant (int) which selects the form-command history variant.
//
// Returns *casefile.FormWorkflow which holds the modelled form-command history.
// Returns error which reports a base history too short for the selected variant, or nil
// on success.
func formHistory(first, second string, toggleSecond bool, variant int) (*casefile.FormWorkflow, error) {
	if variant == formVariantDuplicate {
		second = first
	}
	workflow := casefile.NewFormWorkflow(first, second, toggleSecond)
	const baseSteps = 8
	if len(workflow.Steps) < baseSteps {
		return nil, fmt.Errorf("form workflow base history has %d steps, want at least %d", len(workflow.Steps), baseSteps)
	}
	switch variant {
	case formVariantExtraRow:

		workflow.Steps = append(workflow.Steps, formCommand("create", first, 0),
			formCommand("toggle", "", formExtraRowID), formCommand("delete", "", formExtraRowID))
	case formVariantDuplicate:

		workflow.Steps = slices.Insert(workflow.Steps, formDuplicateAt, workflow.Steps[formDuplicateAt-1])
	case formVariantReset:

		workflow.Steps = slices.Insert(workflow.Steps, formResetAt, formCommand("reset", "", 0),
			formCommand("invalid", "", 0), formCommand("create", second, 0))
		workflow.Steps = append(workflow.Steps, formCommand("delete", "", formExtraRowID))
	}
	return workflow, nil
}

// addFormWorkflow adds source and expected observations for a persisted form workflow.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes p (Project) which holds the drawn application model.
//
// Returns error which reports an unrepresentable form history, or nil on success.
func addFormWorkflow(c *casefile.Case, p Project) error {
	first := fmt.Sprintf("%s %d A", p.Title, p.Count)
	second := fmt.Sprintf("%s %d B", p.Title, p.Count)
	c.Expected.AdditionalComponents = append(c.Expected.AdditionalComponents, "workflow-field")

	c.Files["pages/main.pk"] = strings.Replace(c.Files["pages/main.pk"], "</main>", "<workflow-field hidden></workflow-field></main>", 1)
	workflow, err := formHistory(first, second, p.Show, p.formHistory)
	if err != nil {
		return err
	}
	c.Expected.FormWorkflow = workflow
	c.Features = append(c.Features, "form-associated-control", "sqlite-action-partial", "seagull-sql-accessors")
	c.Files["pages/workflow.pk"] = formWorkflowPage
	c.Files["partials/task-list.pk"] = formWorkflowList
	c.Files["components/workflow-field.pkc"] = formWorkflowField
	c.Files["pkg/workflowdata/data.go"] = formWorkflowStore
	c.Files["db/migrations/001_tasks.up.sql"] = "CREATE TABLE tasks (id INTEGER PRIMARY KEY AUTOINCREMENT, label TEXT NOT NULL, done INTEGER NOT NULL DEFAULT 0 CHECK (done IN (0,1)));\n"
	c.Files["db/migrations/001_tasks.down.sql"] = "DROP TABLE tasks;\n"
	c.Files["db/queries/tasks.sql"] = `-- piko.query(CreateTask, one)
INSERT INTO tasks(label) VALUES (?) RETURNING id, label, done;

-- piko.query(ListTasks, many)
SELECT id, label, done FROM tasks ORDER BY id ASC;

-- piko.query(GetTask, one)
SELECT id, label, done FROM tasks WHERE id = ?;

-- piko.query(ToggleTask, exec)
UPDATE tasks SET done = 1-done WHERE id = ?;

-- piko.query(DeleteTask, exec)
DELETE FROM tasks WHERE id = ?;
`
	for _, operation := range []string{"Create", "Toggle", "Delete"} {
		field := "ID int64 `json:\"id\"`"
		argument := "input.ID"
		validation := ""
		if operation == "Create" {
			field = "Label string `json:\"label\"`; Repeat []string `json:\"repeat\"`; Ignored string `json:\"ignored\"`"
			argument = "input.Label"
			validation = `if len(input.Repeat) != 2 || input.Repeat[0] != "one" || input.Repeat[1] != "two" || input.Ignored != "" {
		return workflowdata.Task{}, piko.NewValidationError(map[string]string{"repeat": "unexpected successful controls"})
	}`
		}
		c.Files["actions/tasks/"+strings.ToLower(operation)+".go"] = fmt.Sprintf(`package tasks

import (
	"piko.sh/piko"
	"example.org/pikofuzzcase/pkg/workflowdata"
)

type %[1]sInput struct { %[2]s }
type %[1]sAction struct { piko.ActionMetadata }
func (a *%[1]sAction) Call(input %[1]sInput) (workflowdata.Task, error) {
	%[4]s
	return workflowdata.%[1]s(a.Ctx(), %[3]s)
}
`, operation, field, argument, validation)
	}
	c.Probes = append(c.Probes, textProbe("/workflow", "SQLite task list"))
	return nil
}

// formCommand builds one form-command history entry.
//
// Takes op (string) which names the command.
// Takes value (string) which is the submitted label, or empty.
// Takes id (int) which is the targeted row identifier, or zero.
//
// Returns casefile.FormCommand which holds the command.
func formCommand(op, value string, id int) casefile.FormCommand {
	return casefile.FormCommand{Op: op, Value: value, ID: id}
}
