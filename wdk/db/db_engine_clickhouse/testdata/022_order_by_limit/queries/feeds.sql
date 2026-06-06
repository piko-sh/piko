-- piko.query(RecentFeeds, many)
SELECT id, title, published
FROM feeds
ORDER BY published DESC
LIMIT {page_size:UInt32} OFFSET {page_offset:UInt32};
