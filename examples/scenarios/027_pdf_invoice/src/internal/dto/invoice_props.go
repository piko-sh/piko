package dto

// InvoiceProps holds the data passed to the invoice PDF template.
// The prop tags map to template variables accessed via {{ props.CustomerName }} etc.
type InvoiceProps struct {
	InvoiceNo    string `prop:"invoice_no"`
	Date         string `prop:"date"`
	CustomerName string `prop:"customer_name"`
	ItemName     string `prop:"item_name"`
	Quantity     int    `prop:"quantity"`
	UnitPrice    string `prop:"unit_price"`
	Total        string `prop:"total"`
}
