package bot

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/bwmarrin/discordgo"
	"github.com/emilekm/go-prbf2/prism"
	"github.com/prbf2-tools/prism-bot/internal/config"
	"github.com/sethvargo/go-password/password"
)

func (b *PrismBot) accounts() {
	rolesIDs := make(map[string]int, 0)
	for _, role := range b.conf.RCONUsers.Roles {
		rolesIDs[role.ID] = role.Level
	}

	members, err := b.session.GuildMembers(b.conf.Discord.GuildID, "", 1000)
	if err != nil {
		slog.Error(err.Error())
	}

	for _, member := range members {
		roleForAccount := b.memberAccountRole(member)
		if roleForAccount != nil {
			err = b.ensureAccount(member, roleForAccount)
			if err != nil {
				slog.Error(fmt.Sprintf("Error ensuring account for %s: %s", member.DisplayName(), err.Error()))
			}
		} else {
			err = b.ensureNoAccount(member)
			if err != nil {
				slog.Error(fmt.Sprintf("Error ensuring no account for %s: %s", member.DisplayName(), err.Error()))
			}
		}
	}

	b.session.AddHandler(func(s *discordgo.Session, m *discordgo.GuildMemberUpdate) {
		// TODO: leverage m.BeforeUpdate.Roles to avoid the ensureNoAccount call
		roleForAccount := b.memberAccountRole(m.Member)

		if roleForAccount != nil {
			err = b.ensureAccount(m.Member, roleForAccount)
			if err != nil {
				slog.Error(fmt.Sprintf("Error ensuring account for %s: %s", m.Member.DisplayName(), err.Error()))
			}
		} else {
			err = b.ensureNoAccount(m.Member)
			if err != nil {
				slog.Error(fmt.Sprintf("Error ensuring no account for %s: %s", m.Member.DisplayName(), err.Error()))
			}
		}
	})
}

func (b *PrismBot) memberAccountRole(member *discordgo.Member) *config.Role {
	var roleForAccount *config.Role

	for _, role := range b.conf.RCONUsers.Roles {
		if slices.Contains(member.Roles, role.ID) {
			if roleForAccount == nil || role.Level < roleForAccount.Level {
				roleForAccount = &role
			}
		}
	}

	return roleForAccount
}

func (b *PrismBot) ensureAccount(member *discordgo.Member, roleForAccount *config.Role) error {
	account, err := b.hasAccount(member)
	if err != nil {
		return err
	}

	if account == nil {
		err = b.createAccount(member, roleForAccount)
		if err != nil {
			return err
		}
	} else {
		if account.Power != roleForAccount.Level {
			err = b.updateAccount(member, roleForAccount)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (b *PrismBot) ensureNoAccount(member *discordgo.Member) error {
	account, err := b.hasAccount(member)
	if err != nil {
		return err
	}

	if account != nil {
		_, err := b.prism.Users.Delete(context.Background(), member.DisplayName())
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *PrismBot) hasAccount(member *discordgo.Member) (*prism.User, error) {
	users, err := b.prism.Users.List(context.Background())
	if err != nil {
		return nil, err
	}

	for _, user := range users {
		if user.Name == member.DisplayName() {
			return &user, nil
		}
	}

	return nil, nil
}

func (b *PrismBot) createAccount(member *discordgo.Member, role *config.Role) error {
	pass, err := password.Generate(12, 6, 0, true, true)
	if err != nil {
		return err
	}

	_, err = b.prism.Users.Add(context.Background(), prism.AddUser{
		Name:     member.DisplayName(),
		Password: pass,
		Power:    role.Level,
	})
	if err != nil {
		return err
	}

	ch, err := b.session.UserChannelCreate(member.User.ID)
	if err != nil {
		return err
	}

	_, err = b.session.ChannelMessageSend(ch.ID, "Your account has been created. Use the following credentials to login:\n\n```\nUsername: "+member.User.Username+"\nPassword: "+pass+"\n```")
	if err != nil {
		return err
	}

	return nil
}

func (b *PrismBot) updateAccount(member *discordgo.Member, role *config.Role) error {
	changeUser := prism.ChangeUser{
		Name:     member.DisplayName(),
		NewName:  member.DisplayName(),
		NewPower: role.Level,
	}

	_, err := b.prism.Users.Change(context.Background(), changeUser)
	return err
}
