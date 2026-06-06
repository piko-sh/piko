-- piko.query(name: TeamRosters, command: many)
SELECT team_id, array_agg(name) AS member_names FROM team_members GROUP BY team_id;
