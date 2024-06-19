package main

import (
	"context"
	"net"
	"os"

	"github.com/emilekm/go-prbf2/prism"
	"github.com/prbf2-tools/prism-bot/internal/config"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		panic(err)
	}
}

func run(args []string) error {
	conf, err := config.NewConfig(args[0])
	if err != nil {
		return err
	}

	client, err := prism.Dial(net.JoinHostPort(conf.Host, conf.Port))
	if err != nil {
		return err
	}

	defer client.Close()

	err = client.Login(context.TODO(), conf.Username, conf.Password)
	if err != nil {
		return err
	}

	return nil
}
