-- piko.name: UnnestNamesAndPrices
-- piko.command: many
SELECT * FROM UNNEST($1::text[], $2::numeric[]) AS t(name text, price numeric);
