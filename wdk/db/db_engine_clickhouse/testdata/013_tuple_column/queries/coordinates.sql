-- piko.query(GetCoordinate, one)
SELECT id, point, named_point FROM coordinates WHERE id = {id:UInt64};
