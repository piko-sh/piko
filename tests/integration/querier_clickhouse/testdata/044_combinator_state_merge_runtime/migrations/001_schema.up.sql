CREATE TABLE source_visits (
    visit_day Date,
    visitor String
) ENGINE = MergeTree() ORDER BY visit_day;

CREATE TABLE daily_uniques (
    visit_day Date,
    visitor_state AggregateFunction(uniq, String)
) ENGINE = AggregatingMergeTree() ORDER BY visit_day;
