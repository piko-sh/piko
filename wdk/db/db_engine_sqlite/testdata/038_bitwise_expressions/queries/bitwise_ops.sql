-- piko.name: GetBitwiseResults
-- piko.command: many
SELECT
  id,
  mask & bits AS masked,
  mask | bits AS combined,
  bits << 2 AS shifted_left,
  bits >> 1 AS shifted_right
FROM flags WHERE id = ?
