-- piko.name: GetTypedData
-- piko.command: one
SELECT id, data, tags, metadata, ip_addr, unique_id, price, raw_data FROM typed_data WHERE id = $1;
