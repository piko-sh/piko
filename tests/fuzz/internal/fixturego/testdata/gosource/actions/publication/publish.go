package publication

import (
	"sync"

	"piko.sh/piko"
)

type PublishInput struct{ Headline string }
type PublicationResult struct{ By, Headline string }
type PublicationLedger struct {
	Calls, Writes   int
	Owner, Headline string
}

var ledgerMu sync.Mutex
var ledger PublicationLedger

type PublishAction struct{ piko.ActionMetadata }

func canEdit(who piko.AuthContext) bool {
	return who != nil && who.IsAuthenticated() && who.Get("publicationRole") == "editor"
}

func (a *PublishAction) Call(in PublishInput) (PublicationResult, error) {
	ledgerMu.Lock()
	defer ledgerMu.Unlock()
	ledger.Calls++
	who := a.Auth()
	if who == nil || !who.IsAuthenticated() {
		return PublicationResult{}, piko.Unauthorised("sign in")
	}
	if !canEdit(who) {
		return PublicationResult{}, piko.Forbidden("only editors publish editions")
	}
	ledger.Writes++
	ledger.Owner = who.UserID()
	ledger.Headline = in.Headline
	return PublicationResult{By: ledger.Owner, Headline: ledger.Headline}, nil
}
