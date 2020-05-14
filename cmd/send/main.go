package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"

	. "github.com/cuappdev/send/internal"
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
				Name:  "logout",
				Usage: "Logout of your account",
				Action: func(c *cli.Context) error {
					ClearCurrentUser()
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
					SendToSlack(fmt.Sprintf("User %s just signed up.", username))
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
						username := GetCurrentUser()
						if GetUser(username).IsAdmin {
							user := c.Args().Get(0)
							app := c.Args().Get(1)
							AddApp(user, app)

							fmt.Printf("Granted user %s access to %s\n", user, app)
							SendToSlack(fmt.Sprintf("User %s granted user %s access to %s.", username, user, app))
						} else {
							fmt.Println("You do not have admin access.")
						}
					}
					return nil
				},
			},
			{
				Name:      "pull",
				Usage:     "Pull the config for an app into the \"config\" directory",
				UsageText: "send pull [APP]",
				Action: func(c *cli.Context) error {
					if c.NArg() < 1 {
						fmt.Println(`"send pull" requires exactly 1 arguments.`)
						cli.ShowCommandHelp(c, c.Command.Name)
					} else {
						app := c.Args().Get(0)

						success := GetAppConfiguration(app)
						if success {
							fmt.Printf("Downloaded successfully the configuration for %q", app)
						} else {
							fmt.Printf("Something went wrong while downloading the configuration for %q", app)
						}
					}
					return nil
				},
			},
			{
				Name:      "push",
				Usage:     "Push a config file for an app. If the file exists on GitHub, it will be updated according. Otherwise, a new file will be created on GitHub.",
				UsageText: "send push [APP] [FILE_PATH]",
				Action: func(c *cli.Context) error {
					if c.NArg() < 2 {
						fmt.Println(`"send push" requires exactly 2 arguments.`)
						cli.ShowCommandHelp(c, c.Command.Name)
					} else {
						app := c.Args().Get(0)
						filePath := c.Args().Get(1)
						username := GetCurrentUser()

						if username == "" {
							return cli.Exit("Login required", 1)
						} else {
							if HasAccessTo(username, app) {
								PushAppConfiguration(username, app, filePath)

								fileName := filepath.Base(filePath)

								fmt.Println(fmt.Sprintf("Pushed %s for %s", fileName, app))
								SendToSlack(fmt.Sprintf("User %s pushed %s for %s", username, fileName, app))
							} else {
								fmt.Println("You don't have access to the specified app.")
							}
						}
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
						app := c.Args().Get(0)
						cmd := c.Args().Tail()
						username := GetCurrentUser()

						if username == "" {
							return cli.Exit("Login required", 1)
						} else {
							if HasAccessTo(username, app) {
								fmt.Println(ExecCmd(app, strings.Join(cmd, " ")))
							} else {
								fmt.Println("You don't have access to the specified app.")
							}
						}

					}
					return nil
				},
			},
			{
				Name:      "provision",
				Usage:     "Creates a new server on DigitalOcean, generates config files, and runs Swarm CLI to setup new server correctly.",
				UsageText: "send provision [FLAGS] [APP]",
				Flags: []cli.Flag{&cli.StringFlag{
					Name:  "size",
					Value: "s-1vcpu-1gb",
					Usage: "To specify the size of the DigitalOcean droplet to be created. Valid sizes include: \n\t" + strings.Join(GetValidSizeStrings(), "\n\t"),
				}},
				Action: func(c *cli.Context) error {
					if c.NArg() < 1 {
						fmt.Println(`"send provision" requires exactly 1 arguments.`)
						cli.ShowCommandHelp(c, c.Command.Name)
					} else {
						username := GetCurrentUser()
						app := c.Args().First()

						if GetUser(username).IsAdmin {
							if !IsDropletSizeValid(c.String("size")) {
								fmt.Println("The specified droplet size is invalid.")
								cli.ShowCommandHelpAndExit(c, c.Command.Name, 1)
							}
							ProvisionServerForApp(app, c.String("size"))
							AddApp(username, app)
							SendToSlack(fmt.Sprintf("User %s provisioned a new server for %s.", username, app))
						} else {
							fmt.Println("You do not have admin access.")
						}
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
