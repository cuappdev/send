package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:  "login",
				Usage: "Login to an account",
				Action: func(c *cli.Context) error {
					username, password := Login()
					_, success := VerifyUser(username, password)
					if success {
						fmt.Println("\nLogin Succeeded")
						WriteUser(username)
					} else {
						fmt.Println("\nUsername doesn't exist or password is incorrect")
					}
					return nil
				},
			},
			{
				Name:  "ls",
				Usage: "List the apps this account has access to",
				Action: func(c *cli.Context) error {
					username := GetCurrentUser()
					if username == "" {
						return cli.Exit("Login required", 1)
					} else {
						apps := GetUser(username).Apps
						if len(apps) == 0 {
							fmt.Println("You don't have access to any apps.")
						} else {
							fmt.Println("You have access to the following apps: " + strings.Join(apps, ", "))
						}
					}
					return nil
				},
			},
			{
				Name:  "signup",
				Usage: "Create an account",
				Action: func(c *cli.Context) error {
					username, password := Signup()
					RegisterUser(username, password)
					fmt.Println("\nNew user registered with username " + username)
					return nil
				},
			},
			{
				Name:      "add",
				Usage:     "Grant a given user access to an app",
				UsageText: "send add [USERNAME] [APP]",
				Action: func(c *cli.Context) error {
					if c.NArg() < 2 {
						fmt.Println(`"send add" requires exactly 2 argument.`)
						cli.ShowCommandHelp(c, c.Command.Name)
					} else {
						fmt.Printf("add %q %q was called", c.Args().Get(0), c.Args().Get(1))
					}
					return nil
				},
			},
			{
				Name:      "pull",
				Usage:     "Pull the config for an app",
				UsageText: "send pull [APP] [DOWNLOAD_PATH]",
				Action: func(c *cli.Context) error {
					if c.NArg() < 2 {
						fmt.Println(`"send pull" requires exactly 2 arguments.`)
						cli.ShowCommandHelp(c, c.Command.Name)
					} else {
						app := c.Args().Get(0)
						path := c.Args().Get(1)

						success := GetAppConfiguration(app, path)
						if success {
							fmt.Printf("Downloaded successfully the configuration for %q", app)
						}
					}
					return nil
				},
			},
			{
				Name:      "push",
				Usage:     "Push the config for an app",
				UsageText: "send push [APP] [CONFIG_PATH]",
				Action: func(c *cli.Context) error {
					if c.NArg() < 2 {
						fmt.Println(`"send push" requires exactly 2 arguments.`)
						cli.ShowCommandHelp(c, c.Command.Name)
					} else {
						fmt.Printf("push %q %q was called", c.Args().Get(0), c.Args().Get(1))
					}
					return nil
				},
			},
			{
				Name:      "exec",
				Usage:     "Run a docker command on an app's deployment",
				UsageText: "send exec [APP] [DOCKER_CMD]",
				Action: func(c *cli.Context) error {
					if c.NArg() < 2 {
						fmt.Println(`"send exec" requires exactly 2 arguments.`)
						cli.ShowCommandHelp(c, c.Command.Name)
					} else {
						fmt.Printf("exec %q %q was called", c.Args().Get(0), c.Args().Get(1))
					}
					return nil
				},
			},
			{
				Name:      "slack",
				Usage:     "Send a message to #send-updates channel on Slack",
				UsageText: "send slack [MESSAGE]",
				Action: func(c *cli.Context) error {
					if c.NArg() < 1 {
						fmt.Println(`"send slack" requires exactly 1 arguments.`)
						cli.ShowCommandHelp(c, c.Command.Name)
					} else {
						SendToSlack(c.Args().Get(0))
						fmt.Printf("slack %q was called", c.Args().Get(0))
					}
					return nil
				},
			},
		},
	}

	app.Name = "Send CLI"
	app.Usage = "A CLI for interfacing with AppDev's deployments"
	app.Version = "1.0.0"

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
