package bot

import (
	"log/slog"
	"strings"
	"text/template"

	"github.com/bwmarrin/discordgo"
	"github.com/emilekm/go-prbf2/prism"
	"github.com/prbf2-tools/prism-bot/internal/config"
	"github.com/prbf2-tools/prism-bot/internal/discord"
)

type PrismBot struct {
	conf    *config.Config
	prism   *prism.Client
	session *discordgo.Session
}

func New(conf *config.Config, prismClient *prism.Client) (*PrismBot, error) {
	return &PrismBot{
		conf:  conf,
		prism: prismClient,
	}, nil
}

func (b *PrismBot) Register(client *discord.Bot) {
	b.session = client.Session()
	go func() {
		slog.Info("Starting to handle messages")
		b.handleMessages()
	}()
}

func (p *PrismBot) handleMessages() {
	for {
		rawMsg, err := p.prism.ReadMessage()
		if err != nil {
			slog.Error(err.Error())
			continue
		}

		switch rawMsg.Subject() {
		case prism.SubjectUpdateServerDetails:
			msg, err := unmarshalMessage[prism.ServerDetails](rawMsg.Body())
			if err != nil {
				slog.Error(err.Error())
				continue
			}

			p.updateServerDetails(msg)
		}
	}
}

func (p *PrismBot) updateServerDetails(msg *prism.ServerDetails) {
	if p.session == nil {
		slog.Error("Discord session is nil")
		return
	}

	for _, channel := range p.conf.ServerDetails.Channels {
		tmpl, err := template.New("serverdetails").Parse(channel.Template)
		if err != nil {
			slog.Error(err.Error())
			return
		}

		var tpl strings.Builder
		err = tmpl.Execute(&tpl, msg)
		if err != nil {
			slog.Error(err.Error())
			return
		}

		_, err = p.session.ChannelEdit(channel.ID, &discordgo.ChannelEdit{
			Name: tpl.String(),
		})
	}
}

func unmarshalMessage[T any](content []byte) (*T, error) {
	var msg T
	err := prism.UnmarshalMessage(content, &msg)
	if err != nil {
		return nil, err
	}

	return &msg, nil
}
