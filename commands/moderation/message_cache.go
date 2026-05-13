package moderation

import (
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

type CachedMessage struct {
	ID      string
	Guild   *discordgo.Guild
	Channel *discordgo.Channel
	Author  *discordgo.User

	Content     string
	Attachments []*discordgo.MessageAttachment
	CreatedAt   time.Time
	Deleted     bool
}

type MessageCache struct {
	messages map[string][]*CachedMessage
	mutex    sync.RWMutex
	limit    int
}

func NewMessageCache(limit int) *MessageCache {
	return &MessageCache{
		messages: make(map[string][]*CachedMessage),
		limit:    limit,
	}
}

func (c *MessageCache) Add(msg *CachedMessage) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	channel := c.messages[msg.Channel.ID]

	channel = append(channel, msg)

	if len(channel) > c.limit {
		channel = channel[len(channel)-c.limit:]
	}

	c.messages[msg.Channel.ID] = channel
}

func (c *MessageCache) Delete(channelID, messageID string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, msg := range c.messages[channelID] {
		if msg.ID == messageID {
			msg.Deleted = true
			return
		}
	}
}

func (c *MessageCache) GetMessages(channelID string) []*CachedMessage {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.messages[channelID]
}
