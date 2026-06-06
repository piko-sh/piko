-- piko.query(GetAccount, one)
SELECT id, email, label, last_login FROM accounts WHERE id = {id:UInt64};
