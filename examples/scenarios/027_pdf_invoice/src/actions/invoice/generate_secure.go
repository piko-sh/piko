package invoice

import (
	"encoding/base64"
	"fmt"
	"time"

	"piko.sh/piko"
	"piko.sh/piko/wdk/pdf"

	"testmodule/internal/dto"
)

// GenerateSecureAction renders an AES-256 encrypted, watermarked PDF invoice.
//
//	action.invoice.GenerateSecure($form)
type GenerateSecureAction struct {
	piko.ActionMetadata
}

// GenerateSecureInput defines the expected form fields. Includes a password
// for the PDF encryption.
type GenerateSecureInput struct {
	CustomerName string `json:"customerName" validate:"required,min=1,max=200"`
	ItemName     string `json:"itemName" validate:"required,min=1,max=200"`
	Quantity     int    `json:"quantity" validate:"required,min=1"`
	UnitPrice    string `json:"unitPrice" validate:"required"`
	Password     string `json:"password" validate:"required,min=4"`
}

// GenerateSecureResponse carries the encrypted PDF back to the client.
type GenerateSecureResponse struct {
	PDF       string `json:"pdf"`
	InvoiceNo string `json:"invoiceNo"`
}

// Call renders the invoice PDF with AES-256 encryption and a watermark.
func (a GenerateSecureAction) Call(input GenerateSecureInput) (GenerateSecureResponse, error) {
	invoice_no := fmt.Sprintf("SEC-%s", time.Now().Format("20060102-150405"))
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

	// Build the transformer registry with the encryption transformer.
	registry := pdf.NewTransformerRegistry()
	if err := registry.Register(pdf.NewEncryptTransformer()); err != nil {
		return GenerateSecureResponse{}, fmt.Errorf("registering encrypt transformer: %w", err)
	}

	transform_config := pdf.TransformConfig{
		EnabledTransformers: []string{"pdf-encrypt"},
		TransformerOptions: map[string]any{
			"pdf-encrypt": pdf.EncryptionOptions{
				Algorithm:     "aes-256",
				OwnerPassword: "piko-owner-" + invoice_no,
				UserPassword:  input.Password,
				Permissions:   0xFFFFF0C4,
			},
		},
	}

	builder, err := pdf.NewRenderBuilderFromDefault()
	if err != nil {
		return GenerateSecureResponse{}, err
	}

	result, err := builder.
		Template("pdfs/secure_invoice.pk").
		Props(props).
		Metadata(pdf.Metadata{
			Title:  "Secure Invoice " + invoice_no,
			Author: "Piko PDF Demo",
		}).
		Watermark("CONFIDENTIAL").
		Transformations(registry, transform_config).
		Do(a.Ctx())
	if err != nil {
		return GenerateSecureResponse{}, err
	}

	return GenerateSecureResponse{
		PDF:       base64.StdEncoding.EncodeToString(result.Content),
		InvoiceNo: invoice_no,
	}, nil
}
