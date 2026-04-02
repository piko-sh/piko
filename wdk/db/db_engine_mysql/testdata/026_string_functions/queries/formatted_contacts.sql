-- piko.name: FormattedContacts
-- piko.command: many
SELECT
    id,
    CONCAT(first_name, ' ', last_name) AS full_name,
    UPPER(last_name) AS upper_last,
    CHAR_LENGTH(first_name) AS name_length,
    SUBSTRING(phone, 1, 3) AS area_code
FROM contacts;
