-- piko.name: GetContactAddresses
-- piko.command: many
SELECT a.street, a.city, a.postcode FROM expand_addresses($1::integer) AS a;
