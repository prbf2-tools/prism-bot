package users

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/bwmarrin/discordgo"
	"github.com/emilekm/go-prbf2/prism"
	"github.com/prbf2-tools/prism-bot/internal/config"
	"github.com/prbf2-tools/prism-bot/internal/discord"
	"github.com/sethvargo/go-password/password"
)

type UsersBot struct {
	prism   *prism.Client
	users   *config.RCONUsers
	guildID string
}

func New(prism *prism.Client, users *config.RCONUsers, guildID string) *UsersBot {
	return &UsersBot{
		prism:   prism,
		users:   users,
		guildID: guildID,
	}
}

func (u *UsersBot) Register(client *discord.Bot) {
	session := client.Session()

	session.AddHandler(func(s *discordgo.Session, _ *discordgo.Ready) {
		u.resolveCurrentRoles(s)
	})

	session.AddHandler(func(s *discordgo.Session, m *discordgo.GuildMemberUpdate) {
		// TODO: leverage m.BeforeUpdate.Roles to avoid the ensureNoAccount call
		u.resolveMemberRoles(s, m.Member)
	})
}

func (u *UsersBot) resolveCurrentRoles(session *discordgo.Session) {
	rolesIDs := make(map[string]int, 0)
	for _, role := range u.users.Roles {
		rolesIDs[role.ID] = role.Level
	}

	// TODO: handle pagination
	members, err := session.GuildMembers(u.guildID, "", 1000)
	if err != nil {
		slog.Error(err.Error())
	}

	for _, member := range members {
		u.resolveMemberRoles(session, member)
	}
}

func (u *UsersBot) resolveMemberRoles(s *discordgo.Session, m *discordgo.Member) {
	roleForAccount := u.memberAccountRole(m)

	if roleForAccount != nil {
		err := u.ensureAccount(s, m, roleForAccount)
		if err != nil {
			slog.Error(fmt.Sprintf("Error ensuring account for %s: %s", m.DisplayName(), err.Error()))
		}
	} else {
		err := u.ensureNoAccount(s, m)
		if err != nil {
			slog.Error(fmt.Sprintf("Error ensuring no account for %s: %s", m.DisplayName(), err.Error()))
		}
	}
}

func (u *UsersBot) memberAccountRole(member *discordgo.Member) *config.Role {
	var roleForAccount *config.Role

	for _, role := range u.users.Roles {
		if slices.Contains(member.Roles, role.ID) {
			if roleForAccount == nil || role.Level < roleForAccount.Level {
				roleForAccount = &role
			}
		}
	}

	return roleForAccount
}

func (u *UsersBot) ensureAccount(s *discordgo.Session, member *discordgo.Member, roleForAccount *config.Role) error {
	account, err := u.hasAccount(member)
	if err != nil {
		return err
	}

	if account == nil {
		err = u.createAccount(s, member, roleForAccount)
		if err != nil {
			return err
		}
	} else {
		if account.Power != roleForAccount.Level {
			err = u.updateAccount(member, roleForAccount)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (u *UsersBot) ensureNoAccount(_ *discordgo.Session, member *discordgo.Member) error {
	account, err := u.hasAccount(member)
	if err != nil {
		return err
	}

	if account != nil {
		_, err := u.prism.Users.Delete(context.Background(), member.DisplayName())
		if err != nil {
			return err
		}
	}

	return nil
}

func (u *UsersBot) hasAccount(member *discordgo.Member) (*prism.User, error) {
	users, err := u.prism.Users.List(context.Background())
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

func (u *UsersBot) createAccount(s *discordgo.Session, member *discordgo.Member, role *config.Role) error {
	pass, err := password.Generate(12, 6, 0, true, true)
	if err != nil {
		return err
	}

	_, err = u.prism.Users.Add(context.Background(), prism.AddUser{
		Name:     member.DisplayName(),
		Password: pass,
		Power:    role.Level,
	})
	if err != nil {
		return err
	}

	ch, err := s.UserChannelCreate(member.User.ID)
	if err != nil {
		return err
	}

	_, err = s.ChannelMessageSend(ch.ID, "Your account has been created. Use the following credentials to login:\n\n```\nUsername: "+member.User.Username+"\nPassword: "+pass+"\n```")
	if err != nil {
		return err
	}

	return nil
}

func (u *UsersBot) updateAccount(member *discordgo.Member, role *config.Role) error {
	changeUser := prism.ChangeUser{
		Name:     member.DisplayName(),
		NewName:  member.DisplayName(),
		NewPower: role.Level,
	}

	_, err := u.prism.Users.Change(context.Background(), changeUser)
	return err
}
