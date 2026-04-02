-- piko.name: GetPromotedValue
-- piko.command: one
SELECT COALESCE(small_value, large_value) as promoted FROM measurements WHERE id = $1
