package functions

import (
	"reflect"
	"strconv"
	"strings"
	"time"

	"slices"

	"github.com/RhykerWells/Summit/common"
	"github.com/bwmarrin/discordgo"
)

// Guild functions

// GetGuild returns the full guild object for a guild
func GetGuild(guildID string) *discordgo.Guild {
	guild, _ := common.Session.Guild(guildID)
	return guild
}

// GuildKickMember kicks a user from the current guild
func GuildKickMember(guildID, userID, reason string) error {
	err := common.Session.GuildMemberDeleteWithReason(guildID, userID, reason)
	return err
}

// GuildBanMember bans a user from the current guild
func GuildBanMember(guildID, userID, reason string) error {
	err := common.Session.GuildBanCreateWithReason(guildID, userID, reason, 0)
	return err
}

// GuildUnbanMember unbans a member from the current guild
func GuildUnbanMember(guildID, userID string) error {
	err := common.Session.GuildBanDelete(guildID, userID)
	return err
}

// Message functions

// SendBasicMessage sends a string as message content to the given channel
// If a delay is included, then the message is deleted after X seconds.
func SendBasicMessage(channelID string, message string, delay ...time.Duration) (msg *discordgo.Message, err error) {
	msg, err = common.Session.ChannelMessageSend(channelID, message)
	if err != nil {
		return nil, err
	}
	if len(delay) > 0 {
		DeleteMessage(channelID, msg.ID, delay[0])
	}
	return msg, nil
}

// SendMessage sends complex message objects to a given channel. Supporting, embed, components etc.
// If a delay is included, then the message is deleted after X seconds
func SendMessage(channelID string, message *discordgo.MessageSend, delay ...time.Duration) (msg *discordgo.Message, err error) {
	msg, err = common.Session.ChannelMessageSendComplex(channelID, message)
	if err != nil {
		return nil, err
	}
	if len(delay) > 0 {
		DeleteMessage(channelID, msg.ID, delay[0])
	}
	return msg, nil
}

// SendDM sends complex message objects to a given users DM channel. Supporting, embed, components etc.
func SendDM(userID string, message *discordgo.MessageSend) error {
	channel, err := common.Session.UserChannelCreate(userID)
	if err != nil {
		return err
	}
	_, err = SendMessage(channel.ID, message)

	return err
}

// EditBasicMessage edits a 'basic' message and allows replacement of the message content
func EditBasicMessage(channelID, messageID, message string) error {
	_, err := common.Session.ChannelMessageEdit(channelID, messageID, message)
	return err
}

// EditMessage edits a 'complex' message and allows replacement of all message objects
func EditMessage(channelID string, messageID string, message *discordgo.MessageSend) error {
	edit := &discordgo.MessageEdit{
		ID:      messageID,
		Channel: channelID,
	}
	if message.Content != "" {
		edit.Content = &message.Content
	}
	if message.Embed != nil {
		edit.Embed = message.Embed
	}
	if message.Embeds != nil {
		edit.Embeds = &message.Embeds
	}
	if message.Components != nil {
		edit.Components = &message.Components
	}
	_, err := common.Session.ChannelMessageEditComplex(edit)

	return err
}

// DeleteMessage deletes a given message immediately or an option delay
func DeleteMessage(channelID, messageID string, delay ...time.Duration) error {
	var duration time.Duration
	if len(delay) > 0 {
		duration = delay[0]
	}
	time.Sleep(duration)
	err := common.Session.ChannelMessageDelete(channelID, messageID)

	return err
}

// Channel functions

// GetChannel returns the channel object if possible from a channel ID
func GetChannel(guildID, channel string) (*discordgo.Channel, error) {
	if strings.HasPrefix(channel, "<@") {
		channel = channel[2 : len(channel)-1]
	}
	guildChannels, _ := common.Session.GuildChannels(guildID)
	c, err := common.Session.Channel(channel)
	if slices.Contains(guildChannels, c) {
		return c, nil
	}
	return nil, err
}

// User functions

// GetUser returns the user object if possible of a user ID/mention
func GetUser(userStr string) (*discordgo.User, error) {
	// Direct mention
	if strings.HasPrefix(userStr, "<@") {
		userStr = userStr[2 : len(userStr)-1]
		userStr = strings.TrimPrefix(userStr, "!")
	}

	u, err := common.Session.User(userStr)

	return u, err
}

// GetMember returns the member object if possible of a user ID/Mention
func GetMember(guildID string, userStr string) (*discordgo.Member, error) {
	// Direct mention
	if strings.HasPrefix(userStr, "<@") {
		userStr = userStr[2 : len(userStr)-1]
	}

	u, err := common.Session.GuildMember(guildID, userStr)

	return u, err
}

// IsMemberHigher returns true is memberA has a higher highest role than memberB
func IsMemberHigher(guildID string, memberA, memberB *discordgo.Member) bool {
	guild := GetGuild(guildID)
	if memberA.User.ID == guild.OwnerID {
		return true // MemberA is the Owner so automatically true
	} else if memberB.User.ID == guild.OwnerID {
		return false // MemberB is the Owner so automatically false
	}

	memAHighRole := HighestRole(guildID, memberA)
	memBHighRole := HighestRole(guildID, memberB)

	if memAHighRole == nil && memBHighRole == nil {
		return false // Neither have roles
	} else if memAHighRole != nil && memBHighRole == nil {
		return true // member B has no roles, so automatically true
	} else if memAHighRole == nil && memBHighRole != nil {
		return false // member A has no roles, so automatically false
	}

	return IsRoleHigher(memAHighRole, memBHighRole)
}

// Role functions

// GetRole returns the full guild role object for a role ID/mention
func GetRole(guildID, roleStr string) (role *discordgo.Role, err error) {
	guild, err := common.Session.Guild(guildID)
	if err != nil {
		return nil, err
	}

	// Role mention
	if strings.HasPrefix(roleStr, "<@&") {
		roleStr = roleStr[3 : len(roleStr)-1]
	}

	for i := range guild.Roles {
		if guild.Roles[i].ID == roleStr {
			role = guild.Roles[i]
			break
		}
	}

	return role, nil
}

// AddRole adds a given roleID to a user
func AddRole(guildID string, memberID, roleID string) error {
	member, err := GetMember(guildID, memberID)
	if err != nil {
		return err
	}
	if slices.Contains(member.Roles, roleID) {
		// User has the role
		return nil
	}
	err = common.Session.GuildMemberRoleAdd(guildID, memberID, roleID)

	return err
}

// RemoveRole removes a given roleID from a user
func RemoveRole(guildID, memberID, roleID string) error {
	member, err := GetMember(guildID, memberID)
	if err != nil {
		return err
	}
	if !slices.Contains(member.Roles, roleID) {
		// User doesn't have role
		return nil
	}
	err = common.Session.GuildMemberRoleRemove(guildID, memberID, roleID)

	return err
}

// SetRoles flushes a users roles and reassigns them to the given roleIDs
func SetRoles(guildID, memberID string, roleIDs []string) error {
	member, err := GetMember(guildID, memberID)
	if err != nil {
		return err
	}
	roles := make(map[string]struct{})

	for _, id := range member.Roles {
		role, _ := GetRole(guildID, id)
		if role != nil && role.Managed {
			roles[id] = struct{}{}
		}
	}
	roleSlice := make([]string, 0, len(roles))
	for id := range roles {
		roleSlice = append(roleSlice, id)
	}
	userData := &discordgo.GuildMemberParams{
		Roles: &roleSlice,
	}
	_, err = common.Session.GuildMemberEdit(guildID, memberID, userData)

	return err
}

// HighestRole returns the role object of a members highest role
// Will return nil if no role is found
func HighestRole(guildID string, member *discordgo.Member) (role *discordgo.Role) {
	guild := GetGuild(guildID)
	for _, memberRoleID := range member.Roles {
		for _, guildRole := range guild.Roles {
			if memberRoleID != guildRole.ID {
				continue
			}
			if role == nil || IsRoleHigher(guildRole, role) {
				role = guildRole
			}
			break
		}
	}
	return role
}

// IsRoleHigher returns a boolean if the position of role A is higher than role B
// If they are both 1 (denoting a new role), we check against the ID
func IsRoleHigher(higher, lower *discordgo.Role) bool {
	if higher.Position != lower.Position {
		return higher.Position > lower.Position
	}
	if higher.ID == lower.ID {
		// Don't want to allow against ourselves or other similarly ranked users
		return false
	}

	// Failed both checks above. Roles both have a position of 1
	// Returns true if highers role is less then lower
	return higher.ID < lower.ID
}

// Misc
func SetStatus(statusText string) {
	// TODO VERSION on nothing
	if statusText == "" {
		statusText = ""
	}

	common.Session.UpdateCustomStatus(statusText)
}

// Helper tools

// ToInt64 takes the value of an int, float or string and returns it as a whole 64-bit integer if possible or 0 when not.
func ToInt64(conv any) int64 {
	t := reflect.ValueOf(conv)
	switch {
	case t.CanInt():
		return t.Int()
	case t.CanFloat():
		if t.Float() == float64(int64(t.Float())) {
			return int64(t.Float())
		}
		return 0
	case t.Kind() == reflect.String:
		i, _ := strconv.ParseFloat(t.String(), 64)
		return ToInt64(i)
	default:
		return 0
	}
}
