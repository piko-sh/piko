-- piko.query(GetLabelled, one)
SELECT id, metadata, counters FROM labelled WHERE id = {id:UInt64};
