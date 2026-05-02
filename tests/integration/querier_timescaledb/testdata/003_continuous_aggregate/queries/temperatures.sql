-- piko.query(name: InsertTemperature, command: exec)
INSERT INTO temperatures (ts, location_id, temperature) VALUES ($1, $2, $3);

-- piko.query(name: CountTemperatures, command: one)
SELECT count(*) FROM temperatures;
