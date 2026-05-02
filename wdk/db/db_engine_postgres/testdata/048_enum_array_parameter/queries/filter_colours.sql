-- piko.query(name: FilterColours, command: many)
SELECT id, name FROM items WHERE colour = ANY($1::colour[]);
