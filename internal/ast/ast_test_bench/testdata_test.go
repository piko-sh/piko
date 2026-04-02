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

//go:build bench

package ast_test_bench

import (
	"context"
	"fmt"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
)

var (
	SinkAST        *ast_domain.TemplateAST
	SinkNode       *ast_domain.TemplateNode
	SinkExpr       ast_domain.Expression
	SinkTokens     []ast_domain.QueryToken
	SinkSelector   ast_domain.SelectorSet
	SinkNodes      []*ast_domain.TemplateNode
	SinkBool       bool
	SinkInt        int
	SinkLexerToken any
)

const (
	TemplateMinimal = `<div></div>`
	TemplateSimple  = `<div class="simple">Hello, World!</div>`
	TemplateNested  = `
<div><div><div><div><div><div><div><div><div><div>
    <p>Item 1</p><p>Item 2</p><p>Item 3</p><p>Item 4</p><p>Item 5</p>
    <p>Item 6</p><p>Item 7</p><p>Item 8</p><p>Item 9</p><p>Item 10</p>
</div></div></div></div></div></div></div></div></div></div>
`
	TemplateWithAttributes = `
<input type="text" id="username" name="user" class="form-input primary"
       placeholder="Enter username" autocomplete="off" required disabled
       data-validate="true" data-max-length="50" aria-label="Username input">
`
	TemplateComplex = `
<div class="card" :class="{ 'card--active': isActive, 'card--dark': theme == 'dark' }">
    <h1 p-if="post.title">{{ post.title }}</h1>
    <article p-html="post.content"></article>
    <ul p-if="post.comments.length > 0">
        <li p-for="(comment, index) in post.comments" :key="comment.id">
            <span>{{ index + 1 }}:</span>
            <p>{{ comment.body }}</p>
        </li>
    </ul>
    <p p-else>No comments yet.</p>
    <button p-on:click="upvote(post.id)" :disabled="isVoting">Upvote</button>
</div>
`
	TemplateExpressionHeavy = `
<div
    :data-id="product.id * 1000"
    :class="['item', product.category, { 'on-sale': product.price < 50.00 }]"
    p-if="(user.isAdmin || user.isOwner) && product.isAvailable && !product.isArchived"
>
    <h2 p-text="product.name.toUpperCase()"></h2>
    <p p-text="'Price: $' + (product.price * (1 + taxRate)).toFixed(2)"></p>
    <a p-bind:href="'/products/' + product.slug">View Details</a>
    <button p-on:click="addToCart(product.id, 1, { immediate: true })">Add to Cart</button>
</div>
`
	TemplateMega = `
<div class="dashboard-container theme-dark" :class="[user.preferences.theme, { 'is-loading': app.loading }]">
    <header class="main-header" p-if="user.isAuthenticated">
        <div class="user-profile">
            <img :src="user.avatarUrl" p-bind:alt="user.name + ' avatar'">
            <div class="user-details">
                <span p-text="user.name"></span>
                <small p-if="user.isAdmin" class="role-badge">Admin</small>
                <small p-else-if="user.isEditor" class="role-badge">Editor</small>
            </div>
        </div>
        <nav>
            <a href="/dashboard">Home</a>
            <a href="/settings">Settings</a>
            <button p-on:click.prevent="logout()">Logout</button>
        </nav>
    </header>
    <p p-else>Please log in to continue.</p>
    <main class="dashboard-content" p-if="!app.loading && app.hasData">
        <section class="stats-grid">
            <div class="stat-card" p-for="(stat, key) in stats" :key="key">
                <h3 p-text="stat.title"></h3>
                <p class="stat-value">{{ stat.value.toFixed(2) }}</p>
                <p class="stat-change" :class="{ 'positive': stat.change >= 0, 'negative': stat.change < 0 }">
                    {{ stat.change >= 0 ? '+' : '' }}{{ stat.change.toFixed(1) }}%
                </p>
            </div>
        </section>
        <section class="data-table-section" p-if="dataTable.items.length > 0">
            <h2>Recent Orders</h2>
            <table>
                <thead>
                    <tr>
                        <th>ID</th><th>Customer</th><th>Status</th><th>Total</th><th>Date</th><th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    <template p-for="(order, index) in dataTable.items">
                        <tr :class="{ 'is-shipped': order.status == 'shipped', 'is-pending': order.status == 'pending' }">
                            <td>#{{ order.id }}</td>
                            <td>{{ order.customer.name }}</td>
                            <td>{{ order.status.toUpperCase() }}</td>
                            <td>${{ (order.total / 100).toFixed(2) }}</td>
                            <td>{{ formatDate(order.date) }}</td>
                            <td>
                                <button :disabled="!canEdit(order)" p-on:click="editOrder(order.id)">Edit</button>
                                <button p-if="user.isAdmin" p-on:click="deleteOrder(order.id)">Delete</button>
                            </td>
                        </tr>
                    </template>
                </tbody>
            </table>
        </section>
    </main>
    <div class="loading-overlay" p-show="app.loading">
        <p>Loading application data...</p>
    </div>
</div>
`
	ExprIdentifier           = `user`
	ExprMemberAccess         = `user.profile.name`
	ExprIndexAccess          = `items[0].value`
	ExprOptionalChain        = `user?.profile?.settings?.theme`
	ExprFunctionCall         = `formatDate(order.date)`
	ExprFunctionCallMultiArg = `formatCurrency(price, 'USD', { decimals: 2 })`
	ExprBinarySimple         = `a + b`
	ExprBinaryComplex        = `(user.age >= 18 && user.isActive) || hasRole(user, 'admin')`
	ExprTernary              = `isActive ? 'Active' : 'Inactive'`
	ExprTernaryNested        = `status == 'active' ? 'green' : status == 'pending' ? 'yellow' : 'red'`
	ExprArrayLiteral         = `['item1', 'item2', 'item3', item4, computeItem()]`
	ExprObjectLiteral        = `{ name: user.name, age: user.age, isAdmin: hasRole(user, 'admin') }`
	ExprObjectLiteralNested  = `{ user: { name: 'John', settings: { theme: 'dark', notifications: true } }, timestamp: now() }`
	ExprTemplateLiteral      = "`Hello, ${user.name}! You have ${notifications.count} notifications.`"
	ExprForIn                = `(item, index) in items`
	ExprForInFiltered        = `item in items.filter(i => i.active)`
	ExprComplex              = `(user.isAdmin || (user.permissions?.canEdit && resource.ownerId == user.id)) && !resource.isLocked ? performAction(resource.id, { notify: true }) : showError('Access denied')`
	ExprDeepMemberChain      = `app.state.ui.modals.confirmDialog.buttons.primary.onClick`
	ExprMathHeavy            = `((price * quantity) * (1 + taxRate) - discount) / 100 + shippingCost`
	QueryTag                 = `div`
	QueryClass               = `.card`
	QueryID                  = `#main-content`
	QueryTagWithClass        = `div.card`
	QueryMultipleClasses     = `.card.active.highlighted`
	QueryAttribute           = `[data-id]`
	QueryAttributeValue      = `[type="text"]`
	QueryAttributePrefix     = `[href^="https://"]`
	QueryDescendant          = `div p`
	QueryChild               = `ul > li`
	QueryAdjacentSibling     = `h1 + p`
	QueryGeneralSibling      = `h1 ~ p`
	QueryPseudoClass         = `li:first-child`
	QueryPseudoClassNot      = `div:not(.hidden)`
	QueryComplex             = `div.card > header h1.title`
	QueryMultipleSelectors   = `h1, h2, h3, h4, h5, h6`
	QueryVeryComplex         = `main.content > section.posts article.post:not(.draft) > header > h2.title, main.content > section.posts article.post:not(.draft) > .post-body p:first-child`
)

func GenerateGigaTemplate() string {
	const (
		numFeedItems      = 25
		commentDepth      = 3
		commentsPerLevel  = 2
		numTrendingTopics = 30
	)

	var builder strings.Builder

	builder.WriteString(`
<header class="main-header" :class="{ 'with-alerts': notifications.unreadCount > 0 }">
    <h1>Analytics Dashboard</h1>
    <div class="user-info" p-if="user.loggedIn">
        Welcome, <b p-text="user.name"></b>!
        <ul class="notification-dropdown" p-show="ui.showNotifications">
            <li p-for="notif in notifications.items" :key="notif.id" :class="notif.type">
                <a :href="notif.link" p-text="notif.message"></a>
            </li>
        </ul>
    </div>
</header>
<div class="body-wrapper">
<aside class="sidebar" p-if="!ui.sidebarCollapsed">
    <h2>Trending Topics</h2>
    <ul class="trending-list">
`)

	for i := range numTrendingTopics {
		fmt.Fprintf(&builder, `
        <li p-if="trending[%d].isActive">
            <a :href="'/topics/' + trending[%d].slug">
                {{ trending[%d].name }}
                <span class="score" :style="{ color: getScoreColor(trending[%d].score) }">
                    ({{ trending[%d].score.toFixed(1) }})
                </span>
            </a>
        </li>`, i, i, i, i, i)
	}

	builder.WriteString(`
    </ul>
</aside>
<main class="main-content" :class="[ 'view-mode-' + ui.viewMode ]">
    <section class="feed">
`)

	for i := range numFeedItems {
		fmt.Fprintf(&builder, `
<div class="feed-item card" :data-post-id="feed[%d].id" p-if="!feed[%d].isFilteredOut">
    <div class="item-header">
        <img class="avatar" :src="feed[%d].author.avatar + '?size=50'" />
        <strong p-text="feed[%d].author.name"></strong>
        <span class="timestamp">{{ formatRelativeTime(feed[%d].timestamp) }}</span>
    </div>
    <div class="item-body" p-html="feed[%d].processedHtmlContent"></div>
    <div class="item-stats">
        <span>{{ feed[%d].stats.likes }} Likes</span>
        <span>{{ feed[%d].stats.comments }} Comments</span>
    </div>
    <div class="item-comments">
        <h4>Comments</h4>
`, i, i, i, i, i, i, i, i)

		generateCommentBlock(&builder, 0, i, commentDepth, commentsPerLevel)

		builder.WriteString(`
    </div>
</div>
`)
	}

	builder.WriteString(`
    </section>
</main>
</div>
<footer p-if="stats.totals" class="main-footer">
    <div>Total Posts: {{ stats.totals.posts }}</div>
</footer>
`)

	return builder.String()
}

func generateCommentBlock(builder *strings.Builder, depth, itemIndex, maxDepth, commentsPerLevel int) {
	if depth >= maxDepth {
		return
	}

	var pathBuilder strings.Builder
	fmt.Fprintf(&pathBuilder, "feed[%d]", itemIndex)
	for range depth {
		pathBuilder.WriteString(".comments[0]")
	}
	path := pathBuilder.String()

	fmt.Fprintf(builder, `<ul class="comment-level-%d">`, depth)
	for i := range commentsPerLevel {
		currentPath := fmt.Sprintf("%s.comments[%d]", path, i)
		fmt.Fprintf(builder, `
    <li class="comment" p-if="%s">
        <p><b>{{ %s.author.name }}:</b> {{ %s.text }}</p>
        <button p-on:click="replyToComment(%s.id)">Reply</button>
`, currentPath, currentPath, currentPath, currentPath)
		generateCommentBlock(builder, depth+1, itemIndex, maxDepth, commentsPerLevel)
		builder.WriteString(`    </li>`)
	}
	builder.WriteString(`</ul>`)
}

var TemplateGiga = GenerateGigaTemplate()

func MustParseTemplate(source string) *ast_domain.TemplateAST {
	ast, err := ast_domain.ParseAndTransform(context.Background(), source, "benchmark")
	if err != nil {
		panic(fmt.Sprintf("failed to parse template: %v", err))
	}
	return ast
}

var (
	parsedSimple  *ast_domain.TemplateAST
	parsedComplex *ast_domain.TemplateAST
	parsedMega    *ast_domain.TemplateAST
	parsedGiga    *ast_domain.TemplateAST
)

func GetParsedSimple() *ast_domain.TemplateAST {
	if parsedSimple == nil {
		parsedSimple = MustParseTemplate(TemplateSimple)
	}
	return parsedSimple
}

func GetParsedComplex() *ast_domain.TemplateAST {
	if parsedComplex == nil {
		parsedComplex = MustParseTemplate(TemplateComplex)
	}
	return parsedComplex
}

func GetParsedMega() *ast_domain.TemplateAST {
	if parsedMega == nil {
		parsedMega = MustParseTemplate(TemplateMega)
	}
	return parsedMega
}

func GetParsedGiga() *ast_domain.TemplateAST {
	if parsedGiga == nil {
		parsedGiga = MustParseTemplate(TemplateGiga)
	}
	return parsedGiga
}

func CountNodes(ast *ast_domain.TemplateAST) int {
	count := 0
	ast.Walk(func(_ *ast_domain.TemplateNode) bool {
		count++
		return true
	})
	return count
}
