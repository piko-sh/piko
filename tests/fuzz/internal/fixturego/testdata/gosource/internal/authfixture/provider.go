package authfixture

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"piko.sh/piko"
)

type publicationRole string

func (r publicationRole) String() string { return string(r) }

type identity struct {
	id, role, meeting string
}

func (i identity) IsAuthenticated() bool { return true }
func (i identity) UserID() string        { return i.id }
func (i identity) Get(key string) any {
	switch key {
	case "roles":
		return i.role
	case "publicationRole":
		return i.role
	case "meeting":
		return i.meeting
	default:
		return nil
	}
}

type provider struct {
	mu                 sync.Mutex
	arrived            map[string]bool
	both               chan struct{}
	renewalUnavailable bool
}

func (p *provider) Authenticate(ctx context.Context, r *http.Request) (piko.AuthContext, error) {
	cookie, err := r.Cookie("publication_session")
	if err != nil {
		return nil, nil
	}
	var who identity
	switch cookie.Value {
	case "reader-ticket-sample":
		who = identity{id: "subscriber-sample", role: "reader", meeting: "single"}
	case "editor-ticket-sample":
		who = identity{id: "publisher-sample", role: "editor", meeting: "single"}
	default:
		return nil, nil
	}
	if r.URL.Path == "/publication/profile" && r.URL.Query().Get("edition") == "renewal" {
		p.mu.Lock()
		failed := p.renewalUnavailable
		p.renewalUnavailable = false
		p.mu.Unlock()
		if failed {
			return nil, errors.New("fixture subscription service temporarily unavailable")
		}
	}
	if r.URL.Path == "/publication/profile" && r.URL.Query().Get("meeting") == "paired" {
		p.mu.Lock()
		if !p.arrived[who.role] {
			p.arrived[who.role] = true
			if len(p.arrived) == 2 {
				close(p.both)
			}
		}
		p.mu.Unlock()
		timer := time.NewTimer(8 * time.Second)
		defer timer.Stop()
		select {
		case <-p.both:
			who.meeting = "paired"
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
			return nil, errors.New("publication meeting timed out")
		}
	}
	return who, nil
}

func Options() []piko.Option {
	return []piko.Option{
		piko.WithAuthProvider(&provider{arrived: make(map[string]bool), both: make(chan struct{}), renewalUnavailable: true}),
		piko.WithAuthGuard(piko.AuthGuardConfig{
			LoginPath:      "/publication/sign-in",
			RedirectParam:  "next",
			PublicPaths:    []string{"/main"},
			PublicPrefixes: []string{"/_piko/assets", "/_piko/partials", "/_piko/dev", "/assets"},
		}),
	}
}
