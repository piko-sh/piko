-- piko.query(GetNodeOrParent, many)
SELECT id, parent_id FROM nodes WHERE id = {nid:UInt64} OR parent_id = {nid:UInt64};
