package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/emilekm/go-prbf2/prism"
	"github.com/prbf2-tools/prism-bot/internal/bot"
	"github.com/prbf2-tools/prism-bot/internal/bot/users"
	"github.com/prbf2-tools/prism-bot/internal/config"
	"github.com/prbf2-tools/prism-bot/internal/discord"
)

var configFilePath string

func main() {
	flag.StringVar(&configFilePath, "config", "config.yaml", "Path to the configuration file")
	flag.Parse()

	if err := run(os.Args[1:]); err != nil {
		panic(err)
	}
}

func run(_ []string) error {
	conf, err := config.NewConfig(configFilePath)
	if err != nil {
		return err
	}

	fmt.Printf("Starting bot with configuration: %+v\n", conf)

	client, err := prism.Dial(net.JoinHostPort(conf.PRISM.Host, conf.PRISM.Port))
	if err != nil {
		return err
	}

	defer client.Close()

	err = client.Login(context.TODO(), conf.PRISM.Username, conf.PRISM.Password)
	if err != nil {
		return err
	}

	dBot := discord.New(&conf.Discord)

	prismBot, err := bot.New(conf, client)
	if err != nil {
		return err
	}

	prismBot.Register(dBot)

	usersBot := users.New(client, &conf.RCONUsers, conf.Discord.GuildID)
	usersBot.Register(dBot)

	dBot.Run()

	return nil
}
