-- piko.query(GetEvent, one)
SELECT id, ts FROM analytics.events WHERE id = {eid:UInt64};
