-- piko.query(FormatBookings, many)
SELECT id,
       formatDateTime(ts, '%Y-%m-%d') AS booked_on,
       toYear(ts) AS year
FROM bookings;
