package invoice

import (
	"encoding/base64"
	"fmt"
	"time"

	"piko.sh/piko"
	"piko.sh/piko/wdk/pdf"

	"testmodule/internal/dto"
)

// GenerateAction renders a PDF invoice from form input and returns it as
// base64-encoded bytes.
//
//	action.invoice.Generate($form)
type GenerateAction struct {
	piko.ActionMetadata
}

// GenerateInput defines the expected form fields.
type GenerateInput struct {
	CustomerName string `json:"customerName" validate:"required,min=1,max=200"`
	ItemName     string `json:"itemName" validate:"required,min=1,max=200"`
	Quantity     int    `json:"quantity" validate:"required,min=1"`
	UnitPrice    string `json:"unitPrice" validate:"required"`
}

// GenerateResponse carries the rendered PDF back to the client.
type GenerateResponse struct {
	PDF       string `json:"pdf"`
	InvoiceNo string `json:"invoiceNo"`
}

// Call renders the invoice PDF and returns it as a base64 string.
func (a GenerateAction) Call(input GenerateInput) (GenerateResponse, error) {
	invoice_no := fmt.Sprintf("INV-%s", time.Now().Format("20060102-150405"))
	total := fmt.Sprintf("%.2f", float64(input.Quantity)*parsePrice(input.UnitPrice))

	props := dto.InvoiceProps{
		InvoiceNo:    invoice_no,
		Date:         time.Now().Format("2 January 2006"),
		CustomerName: input.CustomerName,
		ItemName:     input.ItemName,
		Quantity:     input.Quantity,
		UnitPrice:    input.UnitPrice,
		Total:        total,
	}

	builder, err := pdf.NewRenderBuilderFromDefault()
	if err != nil {
		return GenerateResponse{}, err
	}

	result, err := builder.
		Template("pdfs/invoice.pk").
		Props(props).
		Metadata(pdf.Metadata{Title: "Invoice " + invoice_no}).
		Do(a.Ctx())
	if err != nil {
		return GenerateResponse{}, err
	}

	return GenerateResponse{
		PDF:       base64.StdEncoding.EncodeToString(result.Content),
		InvoiceNo: invoice_no,
	}, nil
}

// parsePrice converts a price string to float64, defaulting to 0.
func parsePrice(s string) float64 {
	var price float64
	_, _ = fmt.Sscanf(s, "%f", &price)
	return price
}
