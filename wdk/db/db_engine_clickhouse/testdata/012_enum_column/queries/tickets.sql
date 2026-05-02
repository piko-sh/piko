-- piko.query(GetTicket, one)
SELECT id, status, priority FROM tickets WHERE id = {tid:UInt64};
