package economy

var GuildEconomySchema = []string{`
CREATE TABLE IF NOT EXISTS economy_config (
	-- General
	guild_id TEXT PRIMARY KEY,
	economy_enabled BOOL DEFAULT FALSE NOT NULL,
	economy_symbol TEXT NOT NULL DEFAULT '£',
	economy_start_balance BIGINT NOT NULL DEFAULT 200,

	-- Monkey making management
	economy_min_return BIGINT NOT NULL DEFAULT 200,
	economy_max_return BIGINT NOT NULL DEFAULT 500,
	economy_max_bet BIGINT NOT NULL DEFAULT 5000,

	-- Custom responses
	economy_custom_work_responses_enabled BOOL DEFAULT FALSE NOT NULL,
	economy_custom_work_responses TEXT[] DEFAULT '{}' NOT NULL,
	economy_custom_crime_responses_enabled BOOL DEFAULT FALSE NOT NULL,
	economy_custom_crime_responses TEXT[] DEFAULT '{}' NOT NULL
);
`, `

`, `
CREATE TABLE IF NOT EXISTS economy_users (
	guild_id TEXT NOT NULL,
	user_id TEXT NOT NULL,
	cash BIGINT NOT NULL,
	bank BIGINT NOT NULL,
	cfwinchance BIGINT NOT NULL DEFAULT 50,
	PRIMARY KEY (guild_id, user_id),
	CONSTRAINT fk_guild_user FOREIGN KEY (guild_id)
        REFERENCES economy_config (guild_id) ON DELETE CASCADE
);
`, `
CREATE INDEX IF NOT EXISTS idx_guild_users
	ON economy_users (guild_id, user_id);
`, `
CREATE TABLE IF NOT EXISTS economy_user_inventories (
	guild_id TEXT NOT NULL,
	user_id TEXT NOT NULL,
	name TEXT NOT NULL,
	description TEXT NOT NULL,
	quantity BIGINT NOT NULL,
	role TEXT NOT NULL,
	reply TEXT NOT NULL,
	PRIMARY KEY (guild_id, user_id, name),
	CONSTRAINT fk_guild_user_inventory FOREIGN KEY (guild_id)
        REFERENCES economy_config (guild_id) ON DELETE CASCADE
);
`, `
CREATE INDEX IF NOT EXISTS idx_guild_users
	ON economy_users (guild_id, user_id);
`, `
CREATE TABLE IF NOT EXISTS economy_cooldowns (
	guild_id TEXT NOT NULL,
	user_id TEXT NOT NULL,
	type TEXT NOT NULL,
	expires_at TIMESTAMP,
	PRIMARY KEY (guild_id, user_id, type),
	CONSTRAINT fk_guild_user_cooldown FOREIGN KEY (guild_id)
        REFERENCES economy_config (guild_id) ON DELETE CASCADE
);
`, `
CREATE INDEX IF NOT EXISTS idx_guild_user_cooldowns
	ON economy_cooldowns (guild_id, user_id);
`, `
CREATE TABLE IF NOT EXISTS economy_shop (
    guild_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    price BIGINT NOT NULL,
    quantity BIGINT NOT NULL,
    role TEXT NOT NULL,
    reply TEXT NOT NULL,
    soldby TEXT,
    PRIMARY KEY (guild_id, name, soldby),
	CONSTRAINT fk_guild_shop FOREIGN KEY (guild_id)
		REFERENCES economy_config (guild_id) ON DELETE CASCADE
);
`, `
CREATE INDEX IF NOT EXISTS idx_item_name
    ON economy_shop (name)
`, `
CREATE TABLE IF NOT EXISTS economy_createitem (
	guild_id TEXT NOT NULL,
	user_id TEXT NOT NULL,
	name TEXT,
	description TEXT,
	price BIGINT,
	quantity BIGINT,
	role TEXT,
	reply TEXT,
	msg_id TEXT NOT NULL UNIQUE,
	PRIMARY KEY (guild_id, user_id),
	CONSTRAINT fk_guild_createitem FOREIGN KEY (guild_id)
        REFERENCES economy_config (guild_id) ON DELETE CASCADE
);
`,
}
