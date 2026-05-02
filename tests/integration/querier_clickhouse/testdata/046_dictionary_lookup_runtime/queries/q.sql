-- piko.query(InsertShipment, exec)
INSERT INTO shipments (id, country_id) VALUES ({id:UInt64}, {country_id:UInt64});

-- piko.query(ShipmentsWithCountry, many)
SELECT
    id,
    country_id,
    dictGetString('country_dict', 'name', country_id) AS country_name
FROM shipments
ORDER BY id;
