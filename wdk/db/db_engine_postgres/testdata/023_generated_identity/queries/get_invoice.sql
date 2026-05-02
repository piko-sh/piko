-- piko.query(name: GetInvoice, command: one)
SELECT id, ref FROM invoices WHERE id = $1;
