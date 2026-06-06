-- piko.query(InsertEvent, exec)
INSERT INTO events (id, category, amount, user_id) VALUES ({id:UInt64}, {category:String}, {amount:UInt32}, {user_id:UInt64});

-- piko.query(FilteredCounts, one)
SELECT
    countIf(category = 'sale') AS sale_count,
    sumIf(amount, category = 'sale') AS sale_total,
    uniqIf(user_id, category = 'sale') AS sale_users
FROM events;
