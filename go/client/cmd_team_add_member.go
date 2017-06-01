// Copyright 2017 Keybase, Inc. All rights reserved. Use of
// this source code is governed by the included BSD license.

package client

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/keybase/cli"
	"github.com/keybase/client/go/libcmdline"
	"github.com/keybase/client/go/libkb"
	"github.com/keybase/client/go/protocol/keybase1"
)

type CmdTeamAddMember struct {
	libkb.Contextified
	team     string
	username string
	role     keybase1.TeamRole
}

func newCmdTeamAddMember(cl *libcmdline.CommandLine, g *libkb.GlobalContext) cli.Command {
	return cli.Command{
		Name:         "add-member",
		ArgumentHelp: "<team name> --user=<username> --role=<owner|admin|writer|reader>",
		Usage:        "add a user to a team",
		Action: func(c *cli.Context) {
			cmd := &CmdTeamAddMember{Contextified: libkb.NewContextified(g)}
			cl.ChooseCommand(cmd, "add-member", c)
		},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "u, user",
				Usage: "username",
			},
			cli.StringFlag{
				Name:  "e, email",
				Usage: "email address to invite",
			},
			cli.StringFlag{
				Name:  "r, role",
				Usage: "team role (owner, admin, writer, reader)",
			},
		},
	}
}

func (c *CmdTeamAddMember) ParseArgv(ctx *cli.Context) error {
	if len(ctx.Args()) != 1 {
		return errors.New("add-member requires team name argument")
	}
	c.team = ctx.Args()[0]
	if len(ctx.String("email")) > 0 {
		return errors.New("add-member via email address not yet supported")
	}

	c.username = ctx.String("user")
	if len(c.username) == 0 {
		return errors.New("username required via --user flag")
	}
	srole := ctx.String("role")
	if srole == "" {
		return errors.New("team role required via --role flag")
	}

	role, ok := keybase1.TeamRoleMap[strings.ToUpper(srole)]
	if !ok {
		return errors.New("invalid team role, please use owner, admin, writer, or reader")
	}
	c.role = role

	return nil
}

func (c *CmdTeamAddMember) Run() error {
	cli, err := GetTeamsClient(c.G())
	if err != nil {
		return err
	}

	arg := keybase1.TeamAddMemberArg{
		Name:     c.team,
		Username: c.username,
		Role:     c.role,
	}

	if err = cli.TeamAddMember(context.Background(), arg); err != nil {
		return err
	}

	dui := c.G().UI.GetDumbOutputUI()

	// send a chat message to the user telling them they are now a member of the team.
	h := newChatServiceHandler(c.G())
	name := strings.Join([]string{c.username, c.G().Env.GetUsername().String()}, ",")
	body := fmt.Sprintf("Hi %s, I've invited you to a new team, %s.", c.username, c.team)
	sendOpts := sendOptionsV1{
		Channel: ChatChannel{
			Name: name,
		},
		Message: ChatMessage{
			Body: body,
		},
	}
	rep := h.SendV1(context.Background(), sendOpts)
	if rep.Error != nil {
		dui.Printf("Success adding user %s to %s, but had an error sending %s a chat message: %s", c.username, c.team, c.username, rep.Error.Message)
		return nil
	}

	dui.Printf("Success! A keybase chat message has been sent to %s.", c.username)

	return nil
}

func (c *CmdTeamAddMember) GetUsage() libkb.Usage {
	return libkb.Usage{
		Config:    true,
		API:       true,
		KbKeyring: true,
	}
}
