package templates

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"

	"github.com/RhykerWells/Summit/common"
	"github.com/bwmarrin/discordgo"
)

type Context struct {
	Guild   *discordgo.Guild
	Channel *discordgo.Channel
	Member  *discordgo.Member
	User    *discordgo.User
	BotUser *discordgo.User
}

func NewContext(g *discordgo.Guild, c *discordgo.Channel, m *discordgo.Member) *Context {
	ctx := &Context{
		Guild:   g,
		Channel: c,
		Member:  m,
		User:    m.User,

		BotUser: common.Bot,
	}

	return ctx
}

func (ctx *Context) Execute(tmplName string, tmplStr string) (string, error) {
	tmpl, err := template.New(tmplName).Option("missingkey=zero").Parse(tmplStr)
	if err != nil {
		return "", errors.New("Error parsing template: " + err.Error())
	}

	var buf bytes.Buffer

	// Optional: limit output
	lw := &limitWriter{w: &buf, n: 2000}

	err = tmpl.Execute(lw, ctx)
	if err != nil {
		return "", fmt.Errorf("Failed executing template: %w", err)
	}

	return buf.String(), nil
}

func Validate(tmplStr string) error {
	_, err := template.New("validate").Option("missingkey=error").Parse(tmplStr)

	return err
}

type limitWriter struct {
	w io.Writer
	n int
}

func (l *limitWriter) Write(p []byte) (int, error) {
	if len(p) > l.n {
		p = p[:l.n]
	}
	l.n -= len(p)
	return l.w.Write(p)
}
