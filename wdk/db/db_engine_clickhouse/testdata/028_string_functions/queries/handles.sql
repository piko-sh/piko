-- piko.query(FormattedHandle, one)
SELECT id,
       lower(name) AS lower_name,
       length(name) AS name_length,
       concat(name, '@', domain) AS handle
FROM handles
WHERE id = {id:UInt64};
