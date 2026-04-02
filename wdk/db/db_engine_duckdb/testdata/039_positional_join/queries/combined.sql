-- piko.name: GetPositionalCombined
-- piko.command: many
SELECT l.id, l.label, r.id AS right_id, r.score
FROM left_values l POSITIONAL JOIN right_values r;
