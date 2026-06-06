-- piko.query(GetVersioned, one)
SELECT id, payload, version FROM versioned FINAL WHERE id = {id:UInt64};
