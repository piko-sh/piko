-- piko.query(GetPrice, one)
SELECT id, amount, fee, micro_amount FROM prices WHERE id = {id:UInt64};
