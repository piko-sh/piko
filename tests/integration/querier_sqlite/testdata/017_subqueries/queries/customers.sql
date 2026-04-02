-- piko.name: ListCustomersWithMaxInvoice
-- piko.command: many
SELECT
    c.id,
    c.name,
    (SELECT max(i.amount) FROM invoices i WHERE i.customer_id = c.id) AS max_invoice,
    EXISTS(SELECT 1 FROM invoices i WHERE i.customer_id = c.id) AS has_invoices
FROM customers c
ORDER BY c.id;
