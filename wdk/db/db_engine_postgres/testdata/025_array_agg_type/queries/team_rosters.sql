-- piko.name: TeamRosters
-- piko.command: many
SELECT team_id, array_agg(name) AS member_names FROM team_members GROUP BY team_id;
