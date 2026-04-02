-- piko.name: GetDistinctSources
-- piko.command: many
SELECT DISTINCT source FROM log_entries ORDER BY source;
