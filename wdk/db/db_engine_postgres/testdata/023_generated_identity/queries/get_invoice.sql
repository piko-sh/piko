-- piko.name: GetInvoice
-- piko.command: one
SELECT id, ref FROM invoices WHERE id = $1;
