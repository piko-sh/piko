-- piko.query(name: GetLogicResults, command: many)
SELECT
  id,
  NOT active as is_inactive,
  active AND verified as both_true,
  active OR verified as either_true
FROM flags
