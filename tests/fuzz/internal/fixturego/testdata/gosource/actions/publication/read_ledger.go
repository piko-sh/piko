package publication

import "piko.sh/piko"

type ReadLedgerAction struct{ piko.ActionMetadata }

func (a *ReadLedgerAction) Call() (PublicationLedger, error) {
	if !canEdit(a.Auth()) {
		return PublicationLedger{}, piko.Forbidden("only editors read the publication ledger")
	}
	ledgerMu.Lock()
	defer ledgerMu.Unlock()
	return ledger, nil
}
