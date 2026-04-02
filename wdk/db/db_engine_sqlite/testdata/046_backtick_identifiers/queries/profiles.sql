-- piko.name: GetProfile
-- piko.command: one
SELECT `id`, `display_name`, `group` FROM `user_profiles` WHERE `id` = ?;
