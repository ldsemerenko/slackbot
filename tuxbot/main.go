// Copyright 2015 Keybase, Inc. All rights reserved. Use of
// this source code is governed by the included BSD license.

package main

import (
	"bytes"
	"log"
	"os/exec"

	"github.com/keybase/slackbot"
	"github.com/keybase/slackbot/cli"
	"github.com/nlopes/slack"
	"gopkg.in/alecthomas/kingpin.v2"
)

func linuxBuildFunc(channel string, args []string) (string, error) {
	config := slackbot.ReadConfigOrDefault()
	if config.DryRun {
		return "Dry Run: Doing that would run `systemctl --user start keybase.prerelease.service`", nil
	}
	if config.Paused {
		return "I'm paused so I can't do that, but I would have run `systemctl --user start keybase.prerelease.service`", nil
	}

	out, err := exec.Command("systemctl", "--user", "start", "keybase.prerelease.service").CombinedOutput()
	if err != nil {
		journal, _ := exec.Command("journalctl", "--since=today", "--user-unit", "keybase.prerelease.service").CombinedOutput()
		api := slack.New(slackbot.GetTokenFromEnv())
		snippetFile := slack.FileUploadParameters{
			Channels: []string{channel},
			Title:    "failed build output",
			Content:  string(journal),
		}
		api.UploadFile(snippetFile) // ignore errors here for now
		return string(out), err
	} else {
		return "SUCCESS", nil
	}
}

func kingpinTuxbotHandler(channel string, args []string) (string, error) {
	app := kingpin.New("tuxbot", "Command parser for tuxbot")
	app.Terminate(nil)
	stringBuffer := new(bytes.Buffer)
	app.Writer(stringBuffer)

	build := app.Command("build", "Build things")
	buildLinux := build.Command("linux", "Start a linux build")

	cmd, usage, err := cli.Parse(app, args, stringBuffer)
	if usage != "" || err != nil {
		return usage, err
	}

	switch cmd {
	case buildLinux.FullCommand():
		return slackbot.FuncCommand{
			Desc: "Perform a linux build",
			Fn:   linuxBuildFunc,
		}.Run(channel, args)
	}

	return cmd, nil
}

func addCommands(bot *slackbot.Bot) {
	bot.AddCommand("date", slackbot.NewExecCommand("/bin/date", nil, true, "Show the current date"))
	bot.AddCommand("pause", slackbot.NewPauseCommand())
	bot.AddCommand("resume", slackbot.NewResumeCommand())
	bot.AddCommand("config", slackbot.NewListConfigCommand())
	bot.AddCommand("toggle-dryrun", slackbot.ToggleDryRunCommand{})

	bot.AddCommand("build", slackbot.FuncCommand{
		Desc: "Build all the things!",
		Fn:   kingpinTuxbotHandler,
	})
}

func main() {
	bot, err := slackbot.NewBot(slackbot.GetTokenFromEnv())
	if err != nil {
		log.Fatal(err)
	}

	addCommands(bot)

	log.Println("Started keybot")
	bot.Listen()
}
