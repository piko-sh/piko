package shop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"example.org/pikofuzzcase/pkg/invoice"
	"piko.sh/piko"
	"piko.sh/piko/wdk/email"
	"piko.sh/piko/wdk/pdf"
)

type PlaceInput struct {
	Customer, SKU string
	Quantity      int
}
type PlaceAction struct{ piko.ActionMetadata }

var saved struct {
	sync.Mutex
	receipt           invoice.Order
	attempts, commits int
}

func recordCheckoutState() error {
	data, err := json.Marshal(struct {
		Order             invoice.Order
		Attempts, Commits int
	}{saved.receipt, saved.attempts, saved.commits})
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(filepath.Dir(os.Getenv("PIKO_FUZZ_CHECKOUT_OUTBOX")), "checkout-state.json"), data, 0600)
}
func (a *PlaceAction) Call(input PlaceInput) (invoice.Order, error) {
	if input.Customer == "" || input.SKU != "linen" || input.Quantity < 1 || input.Quantity > 10 {
		return invoice.Order{}, piko.NewValidationError(map[string]string{"quantity": "invalid order"})
	}
	saved.Lock()
	defer saved.Unlock()
	saved.attempts++
	if err := recordCheckoutState(); err != nil {
		return invoice.Order{}, err
	}
	if saved.receipt.Line != "" {
		if saved.receipt.Customer != input.Customer || saved.receipt.SKU != input.SKU || saved.receipt.Quantity != input.Quantity {
			return invoice.Order{}, fmt.Errorf("different order for this checkout")
		}
		return saved.receipt, nil
	}
	select {
	case yes := <-release:
		if !yes {
			return invoice.Order{}, piko.NewValidationError(map[string]string{"order": "order declined"})
		}
	case <-a.Ctx().Done():
		return invoice.Order{}, a.Ctx().Err()
	case <-time.After(6 * time.Second):
		return invoice.Order{}, fmt.Errorf("approval timed out")
	}
	order := invoice.Order{Customer: input.Customer, SKU: input.SKU, Quantity: input.Quantity, Unit: 105, Total: input.Quantity * 105}
	order.Line = fmt.Sprintf("%s / %s / %d x %d = %d cents", order.Customer, order.SKU, order.Quantity, order.Unit, order.Total)
	doc, err := pdf.NewRenderBuilderFromDefault()
	if err != nil {
		return invoice.Order{}, err
	}
	result, err := doc.Template("pdfs/order.pk").Props(order).Metadata(pdf.Metadata{Title: "Invoice " + order.SKU}).Do(a.Ctx())
	if err != nil {
		return invoice.Order{}, err
	}
	mail, err := email.NewTemplatedEmailBuilderFromDefault[invoice.Order]()
	if err != nil {
		return invoice.Order{}, err
	}
	err = mail.Provider("checkout-disk").Immediate().From("shop@example.test").To("buyer@example.test").Subject("Order "+order.SKU).
		BodyTemplate("emails/order.pk").Props(order).Attachment("invoice.pdf", "application/pdf", result.Content).Do(a.Ctx())
	if err == nil {
		saved.receipt = order
		saved.commits++
		err = recordCheckoutState()
	}
	return order, err
}
