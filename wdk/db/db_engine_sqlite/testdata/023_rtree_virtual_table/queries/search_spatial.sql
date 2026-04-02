-- piko.name: SearchSpatial
-- piko.command: many
SELECT id FROM spatial_idx WHERE min_x >= ? AND max_x <= ? AND min_y >= ? AND max_y <= ?
