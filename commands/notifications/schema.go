package notifications

var GuildNotificationSchema = []string{`
CREATE TABLE IF NOT EXISTS notifications_config (
	guild_id TEXT PRIMARY KEY,

	join_server_channel TEXT DEFAULT '' NOT NULL,
	join_server_message TEXT DEFAULT '' NOT NULL,

	leave_server_channel TEXT DEFAULT '' NOT NULL,
	leave_server_message TEXT DEFAULT '' NOT NULL
);
`}
