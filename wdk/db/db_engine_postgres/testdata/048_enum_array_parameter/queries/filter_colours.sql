-- piko.name: FilterColours
-- piko.command: many
SELECT id, name FROM items WHERE colour = ANY($1::colour[]);
