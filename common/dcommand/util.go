package dcommand

import "github.com/bwmarrin/discordgo"

var PermissionNames = map[int64]string{
	// Channel
	discordgo.PermissionViewChannel:        "View Channel",
	discordgo.PermissionSendMessages:       "Send Messages",
	discordgo.PermissionSendTTSMessages:    "Send TTS Messages",
	discordgo.PermissionManageMessages:     "Manage Messages",
	discordgo.PermissionEmbedLinks:         "Embed Links",
	discordgo.PermissionAttachFiles:        "Attach Files",
	discordgo.PermissionReadMessageHistory: "Read Message History",
	discordgo.PermissionMentionEveryone:    "Mention Everyone",
	discordgo.PermissionAddReactions:       "Add Reactions",

	// Moderation / Management
	discordgo.PermissionKickMembers:     "Kick Members",
	discordgo.PermissionBanMembers:      "Ban Members",
	discordgo.PermissionModerateMembers: "Timeout Members",
	discordgo.PermissionAdministrator:   "Administrator",
	discordgo.PermissionManageNicknames: "Manage Nicknames",
	discordgo.PermissionManageRoles:     "Manage Roles",
	discordgo.PermissionManageChannels:  "Manage Channels",
	discordgo.PermissionManageGuild:     "Manage Server",
	discordgo.PermissionViewAuditLogs:   "View Audit Log",

	// General
	discordgo.PermissionCreateInstantInvite: "Create Invite",
}

func permissionName(perm int64) string {
	if name, ok := PermissionNames[perm]; ok {
		return name
	}

	return "Unknown Permission"
}
