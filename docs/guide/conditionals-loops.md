---
title: Conditionals & loops
description: Advanced patterns for control flow in Piko templates
nav:
  sidebar:
    section: "guide"
    subsection: "concepts"
    order: 37
---

# Conditionals & loops

Master advanced control flow patterns with `p-if` and `p-for` to build dynamic, efficient templates. For expression syntax (operators, ternary, optional chaining, built-in functions), see [template syntax](/docs/guide/template-syntax).

## Advanced conditionals

### Multiple conditions

```piko
<template>
  <div>
    <!-- AND condition -->
    <p p-if="state.IsLoggedIn && state.HasPermission">
      Welcome, admin!
    </p>

    <!-- OR condition -->
    <p p-if="state.IsOwner || state.IsModerator">
      You can edit this content
    </p>

    <!-- Mixed logical operators -->
    <p p-if="(state.IsPremium || state.IsTrial) && !state.IsExpired">
      Premium features enabled
    </p>

    <!-- Comparison chains -->
    <p p-if="state.Age >= 13 && state.Age < 18">
      Teen account
    </p>
  </div>
</template>
```

### Complex conditional chains

```piko
<template>
  <div class="account-status">
    <div p-if="state.Subscription == 'premium'" class="badge-gold">
      <span>Premium member</span>
      <p>Unlimited access to all features</p>
    </div>
    <div p-else-if="state.Subscription == 'pro'" class="badge-silver">
      <span>Pro member</span>
      <p>Access to pro features</p>
    </div>
    <div p-else-if="state.Subscription == 'basic'" class="badge-bronze">
      <span>Basic member</span>
      <p>Access to basic features</p>
    </div>
    <div p-else-if="state.TrialDaysRemaining > 0" class="badge-trial">
      <span>Trial ({{ state.TrialDaysRemaining }} days left)</span>
      <p>Try premium features for free</p>
    </div>
    <div p-else class="badge-free">
      <span>Free account</span>
      <p>Limited access</p>
    </div>
  </div>
</template>
```

### Nested conditionals

```piko
<template>
  <div>
    <div p-if="state.User != nil">
      <h2>Welcome, {{ state.User.Name }}!</h2>

      <div p-if="state.User.HasNewMessages">
        <p p-if="state.User.MessageCount == 1">
          You have 1 new message
        </p>
        <p p-else>
          You have {{ state.User.MessageCount }} new messages
        </p>
      </div>

      <div p-if="len(state.User.Notifications) > 0">
        <h3>Notifications</h3>
        <!-- Notifications here -->
      </div>
    </div>
    <div p-else>
      <p>Please log in to continue</p>
    </div>
  </div>
</template>
```

### Conditional rendering patterns

**Show/hide toggle**:

```piko
<template>
  <div>
    <button p-on:click="action.toggle_details()">
      {{ state.ShowDetails ? "Hide" : "Show" }} Details
    </button>

    <div p-if="state.ShowDetails" class="details-panel">
      <h3>Additional information</h3>
      <p>{{ state.DetailsText }}</p>
    </div>
  </div>
</template>
```

**Permission-based rendering**:

You can call methods on your Response struct to encapsulate permission logic:

```piko
<template>
  <div class="admin-panel">
    <h1>Dashboard</h1>

    <!-- Only admins see this -->
    <section p-if="state.CanManageUsers">
      <h2>User management</h2>
      <!-- User management UI -->
    </section>

    <!-- Admins and moderators -->
    <section p-if="state.CanModerateContent">
      <h2>Content moderation</h2>
      <!-- Moderation tools -->
    </section>

    <!-- Everyone sees this -->
    <section>
      <h2>Your profile</h2>
      <!-- Profile UI -->
    </section>
  </div>
</template>

<script type="application/x-go">
package main

import "piko.sh/piko"

type Response struct {
    CanManageUsers     bool
    CanModerateContent bool
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    user := getCurrentUser(r)
    return Response{
        CanManageUsers:     user.HasPermission("manage_users"),
        CanModerateContent: user.HasPermission("moderate_content"),
    }, piko.Metadata{}, nil
}
</script>
```

**Feature flags**:

```piko
<template>
  <div>
    <!-- Beta feature -->
    <div p-if="state.DarkModeEnabled">
      <button p-on:click="action.toggle_dark_mode()">
        {{ state.DarkMode ? "Light" : "Dark" }} Mode
      </button>
    </div>

    <!-- A/B test variant -->
    <div p-if="state.Variant == 'A'">
      <button class="btn-blue">Sign up</button>
    </div>
    <div p-else>
      <button class="btn-green">Get started</button>
    </div>
  </div>
</template>
```

## Advanced loops

Piko supports two forms of loop syntax:

- **Simple form**: `item in collection` - iterates over values only
- **Indexed form**: `(index, item) in collection` - includes the index/key

### Loop with index

```piko
<template>
  <div>
    <table>
      <thead>
        <tr>
          <th>#</th>
          <th>Name</th>
          <th>Status</th>
        </tr>
      </thead>
      <tbody>
        <tr p-for="(idx, item) in state.Items">
          <td>{{ idx + 1 }}</td>
          <td>{{ item.Name }}</td>
          <td>{{ item.Status }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
```

### Simple loop (value only)

When you don't need the index, use the simple form:

```piko
<template>
  <ul>
    <li p-for="item in state.Items">
      {{ item.Name }}
    </li>
  </ul>
</template>
```

### Filtering in loops

```piko
<template>
  <div>
    <h2>Active items only</h2>
    <div p-for="(_, item) in state.Items">
      <!-- Only render active items -->
      <div p-if="item.IsActive" class="item-card">
        <h3>{{ item.Name }}</h3>
        <p>{{ item.Description }}</p>
      </div>
    </div>
  </div>
</template>
```

> **Better Approach**: Filter in `Render()` function for better performance:

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    allItems := fetchAllItems()

    // Filter active items
    activeItems := []Item{}
    for _, item := range allItems {
        if item.IsActive {
            activeItems = append(activeItems, item)
        }
    }

    return Response{
        Items: activeItems,
    }, piko.Metadata{}, nil
}
```

```piko
<template>
  <div>
    <h2>Active items</h2>
    <div p-for="(_, item) in state.Items" class="item-card">
      <h3>{{ item.Name }}</h3>
      <p>{{ item.Description }}</p>
    </div>
  </div>
</template>
```

### Nested loops

```piko
<template>
  <div>
    <div p-for="(_, category) in state.Categories" class="category-section">
      <h2>{{ category.Name }}</h2>

      <div p-if="len(category.Items) > 0">
        <div p-for="(_, item) in category.Items" class="item">
          <h3>{{ item.Name }}</h3>
          <p>{{ item.Price }}</p>

          <!-- Nested tags -->
          <div class="tags">
            <span p-for="(_, tag) in item.Tags" class="tag">
              {{ tag }}
            </span>
          </div>
        </div>
      </div>
      <p p-else class="empty">No items in this category</p>
    </div>
  </div>
</template>
```

### Loop with separators

```piko
<template>
  <div>
    <!-- Comma-separated list -->
    <p>
      Authors:
      <span p-for="(idx, author) in state.Authors">
        {{ author.Name }}<span p-if="idx < len(state.Authors) - 1">, </span>
      </span>
    </p>

    <!-- Breadcrumbs -->
    <nav class="breadcrumbs">
      <span p-for="(idx, crumb) in state.Breadcrumbs">
        <a :href="crumb.URL">{{ crumb.Title }}</a>
        <span p-if="idx < len(state.Breadcrumbs) - 1"> / </span>
      </span>
    </nav>
  </div>
</template>
```

### First/last item styling

```piko
<template>
  <div>
    <div p-for="(idx, item) in state.Items"
         :class="'item ' +
                 (idx == 0 ? 'first ' : '') +
                 (idx == len(state.Items) - 1 ? 'last' : '')">
      {{ item.Name }}
    </div>
  </div>
</template>
```

Or with computed classes in Render():

```go
type ItemWithClasses struct {
    Item
    Classes string
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    items := fetchItems()
    itemsWithClasses := make([]ItemWithClasses, len(items))

    for idx, item := range items {
        classes := "item"
        if idx == 0 {
            classes += " first"
        }
        if idx == len(items)-1 {
            classes += " last"
        }

        itemsWithClasses[idx] = ItemWithClasses{
            Item:    item,
            Classes: classes,
        }
    }

    return Response{Items: itemsWithClasses}, piko.Metadata{}, nil
}
```

```piko
<template>
  <div>
    <div p-for="(_, item) in state.Items" :class="item.Classes">
      {{ item.Name }}
    </div>
  </div>
</template>
```

### Map iteration

```piko
<template>
  <div>
    <h2>Settings</h2>
    <dl>
      <div p-for="(key, value) in state.Settings">
        <dt>{{ key }}</dt>
        <dd>{{ value }}</dd>
      </div>
    </dl>
  </div>
</template>

<script type="application/x-go">
package main

import "piko.sh/piko"

type Response struct {
    Settings map[string]string
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    return Response{
        Settings: map[string]string{
            "theme":     "dark",
            "language":  "en",
            "timezone":  "UTC",
            "pageSize":  "20",
        },
    }, piko.Metadata{}, nil
}
</script>
```

## Empty state handling

### Basic empty check

```piko
<template>
  <div>
    <div p-if="len(state.Items) > 0">
      <div p-for="(_, item) in state.Items">
        {{ item.Name }}
      </div>
    </div>
    <div p-else class="empty-state">
      <p>No items found</p>
      <button>Add Your First Item</button>
    </div>
  </div>
</template>
```

### Empty states with multiple conditions

```piko
<template>
  <div class="products-page">
    <h1>Products</h1>

    <!-- Loading state -->
    <div p-if="state.IsLoading" class="loading">
      <p>Loading products...</p>
    </div>

    <!-- Error state -->
    <div p-else-if="state.Error != ''" class="error">
      <p>Error: {{ state.Error }}</p>
      <button p-on:click="action.retry()">Try Again</button>
    </div>

    <!-- Empty state (no results) -->
    <div p-else-if="len(state.Products) == 0" class="empty">
      <h2>No products found</h2>
      <p p-if="state.SearchQuery != ''">
        No results for "{{ state.SearchQuery }}"
      </p>
      <p p-else>
        You haven't added any products yet.
      </p>
      <button p-on:click="action.clear_search()">Clear Search</button>
    </div>

    <!-- Success state (has results) -->
    <div p-else class="products-grid">
      <div p-for="(_, product) in state.Products" class="product-card">
        <h3>{{ product.Name }}</h3>
        <p>{{ product.Price }}</p>
      </div>
    </div>
  </div>
</template>
```

## Performance patterns

### Limit large lists

```piko
<template>
  <div>
    <h2>Recent Activity</h2>
    <!-- Only show first 10 items -->
    <div p-for="(idx, item) in state.RecentItems">
      <div p-if="idx < 10">
        {{ item.Title }}
      </div>
    </div>

    <button p-if="len(state.RecentItems) > 10">
      View All {{ len(state.RecentItems) }} Items
    </button>
  </div>
</template>
```

**Better**: Limit in Render():

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    allItems := fetchRecentItems()

    // Only return first 10
    displayItems := allItems
    if len(allItems) > 10 {
        displayItems = allItems[:10]
    }

    return Response{
        DisplayItems: displayItems,
        TotalCount:   len(allItems),
    }, piko.Metadata{}, nil
}
```

### Avoid deep nesting

```piko
<!-- Hard to read and maintain -->
<div p-if="state.User != nil">
  <div p-if="state.User.Profile != nil">
    <div p-if="state.User.Profile.Settings != nil">
      <div p-if="state.User.Profile.Settings.DarkMode">
        Dark mode enabled
      </div>
    </div>
  </div>
</div>

<!-- Flatten in Render() -->
<div p-if="state.DarkModeEnabled">
  Dark mode enabled
</div>
```

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    user := fetchUser()

    darkModeEnabled := false
    if user != nil && user.Profile != nil && user.Profile.Settings != nil {
        darkModeEnabled = user.Profile.Settings.DarkMode
    }

    return Response{
        DarkModeEnabled: darkModeEnabled,
    }, piko.Metadata{}, nil
}
```

## Common patterns

### Pagination

```piko
<template>
  <div>
    <!-- Items -->
    <div p-for="(_, item) in state.Items" class="item">
      {{ item.Name }}
    </div>

    <!-- Pagination controls -->
    <nav class="pagination">
      <button
        p-if="state.Page > 1"
        p-on:click="`action.goto_page(${state.Page - 1})`"
      >
        Previous
      </button>

      <span>Page {{ state.Page }} of {{ state.TotalPages }}</span>

      <button
        p-if="state.Page < state.TotalPages"
        p-on:click="`action.goto_page(${state.Page + 1})`"
      >
        Next
      </button>
    </nav>
  </div>
</template>
```

### Tabs

```piko
<template>
  <div>
    <!-- Tab buttons -->
    <div class="tabs">
      <button
        p-on:click="action.set_tab('overview')"
        :class="state.ActiveTab == 'overview' ? 'active' : ''"
      >
        Overview
      </button>
      <button
        p-on:click="action.set_tab('details')"
        :class="state.ActiveTab == 'details' ? 'active' : ''"
      >
        Details
      </button>
      <button
        p-on:click="action.set_tab('reviews')"
        :class="state.ActiveTab == 'reviews' ? 'active' : ''"
      >
        Reviews
      </button>
    </div>

    <!-- Tab panels -->
    <div class="tab-content">
      <div p-if="state.ActiveTab == 'overview'">
        Overview content...
      </div>
      <div p-else-if="state.ActiveTab == 'details'">
        Details content...
      </div>
      <div p-else-if="state.ActiveTab == 'reviews'">
        Reviews content...
      </div>
    </div>
  </div>
</template>
```

### Accordion

```piko
<template>
  <div class="accordion">
    <div p-for="(idx, item) in state.Items" class="accordion-item">
      <button
        p-on:click="`action.toggle_item(${idx})`"
        class="accordion-header"
      >
        {{ item.Title }}
        <span>{{ item.IsOpen ? '−' : '+' }}</span>
      </button>

      <div p-if="item.IsOpen" class="accordion-content">
        {{ item.Content }}
      </div>
    </div>
  </div>
</template>
```

## Next steps

- [Directives](/docs/guide/directives) → Complete directive reference
- [Template syntax](/docs/guide/template-syntax) → Expressions and operators
- [Partials](/docs/guide/partials) → Break complex logic into components
