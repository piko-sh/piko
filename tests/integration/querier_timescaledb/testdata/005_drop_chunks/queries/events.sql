-- piko.query(name: InsertChunkEvent, command: exec)
INSERT INTO chunk_events (ts, event_id, payload) VALUES ($1, $2, $3);

-- piko.query(name: CountEvents, command: one)
SELECT count(*) FROM chunk_events;
