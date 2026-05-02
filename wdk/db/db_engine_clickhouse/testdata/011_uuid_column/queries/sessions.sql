-- piko.query(GetSession, one)
SELECT id, user_id, created FROM sessions WHERE id = {sid:UUID};
