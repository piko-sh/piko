# `/actions`: Your server-side logic

This directory contains all your server-side **Actions**: Go functions that can be called securely from your templates in response to user interactions. You'll find two examples already scaffolded in `actions/greeting/`: a `SubmitAction` (POST with input) and a `PrintAction` (GET with no input).

Actions are organised into sub-packages. The package name and struct name together determine how the action is called from a template:

```text
action.greeting.Submit({name: "Alice"})
       ────┬───  ──┬──
       package    struct name minus "Action" suffix
```

This approach keeps your business logic clean, organised, type-safe, and decoupled from your rendering code.

---

## How actions work

Actions are **auto-discovered** by the code generator. You do not need to register them manually; just define the struct and the generator will create the necessary wiring.

Run the generator after adding or modifying actions:

```sh
go run ./cmd/generator/main.go all
```

---

## Creating a new action: a step-by-step guide

Let's walk through the `submit` action that was scaffolded with your project.

### Step 1: Define the action struct

An Action is a Go struct that embeds `piko.ActionMetadata` and defines a `Call` method with typed input and output.

```go title="actions/greeting/submit.go"
package greeting

import "piko.sh/piko"

// SubmitAction handles POST requests with a name input.
//
// Embedding piko.ActionMetadata gives the action access to helper methods:
//   - Ctx()      - the request context (for timeouts, cancellation, tracing)
//   - Request()  - the raw *http.Request (for headers, cookies, IP address)
//   - Response() - the http.ResponseWriter (for setting headers, status codes)
type SubmitAction struct {
    piko.ActionMetadata
}

// SubmitInput defines the data the action expects to receive.
// The json tags map form field names to struct fields.
// The validate tags are processed automatically before Call is invoked.
type SubmitInput struct {
    Name string `json:"name" validate:"required"`
}

// SubmitResponse defines the data returned by the action.
type SubmitResponse struct {
    Greeting string `json:"greeting"`
}

// Call returns a personalised greeting for the given name.
func (a SubmitAction) Call(input SubmitInput) (SubmitResponse, error) {
    return SubmitResponse{
        Greeting: "Hello, " + input.Name + "!",
    }, nil
}
```

### Step 2: Run the generator

After creating or modifying actions, run the generator to update the auto-generated registry:

```sh
go run ./cmd/generator/main.go all
```

This wires your actions into the Piko runtime so they can be called from templates.

### Step 3: Trigger the action from a template

Trigger the action from a `.pk` or `.pkc` file using `p-on` with the `action.` prefix:

```html
<form p-on:submit.prevent="action.greeting.Submit($form)">
  <input name="name" />
  <button type="submit">Submit</button>
</form>
```

When this form is submitted, Piko handles everything:
1. It prevents the default page reload (`.prevent`).
2. It gathers the form data and maps it to `SubmitInput` via the json tags.
3. It validates the input using the validate tags.
4. It calls your `Call` method and returns the `SubmitResponse` to the client.

---

### To learn more

For more details on validation, error handling, caching, rate limiting, SSE streaming, and file uploads, please see the **[official Piko documentation on actions](https://piko.sh/docs/reference/server-actions)**.
