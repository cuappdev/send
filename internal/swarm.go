package internal

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var homeDir, _ = os.UserHomeDir()

func ProvisionServerForApp(app string) {
	fmt.Println("SETTING UP SWARM CLI")
	setupSwarmCLI()

	if getDirectory(app) != nil {
		fmt.Println("App " + app + " already exists. Choose a different name.")
		os.Exit(1)
	}
	os.Mkdir(filepath.Join(homeDir, ".send", app), os.ModePerm)

	fmt.Println("GENERATING SERVER PEM KEYS")
	generatePemKeys(app)

	fmt.Println("CREATING DROPLET ON DIGITALOCEAN")
	dropletId := createDroplet(app)

	fmt.Println("WAITING FOR DROPLET TO GET ASSIGNED AN IP ADDRESS")
	for getDropletStatus(dropletId) != "active" {
		time.Sleep(5 * time.Second)
	}

	fmt.Println("CONSTRUCTING APP BUNDLE FOR SWARM CLI")
	constructBundle(app, getDropletIP(dropletId))
	commitBundle(app)

	fmt.Println("WAITING FOR DROPLET TO FINISH INITIALIZING")
	time.Sleep(30 * time.Second)

	runSwarmOnServer(app)
}

func setupSwarmCLI() {
	path := filepath.Join(homeDir, ".send", "swarm-cli")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		// Swarm already setup
		return
	}

	cloneCmd := exec.Command(
		"git",
		"clone",
		"https://github.com/cuappdev/swarm-cli.git",
		path,
	)

	if err := cloneCmd.Run(); err != nil {
		fmt.Printf("Error cloning swarm cli: %s", err)
		os.Exit(1)
	}

	commands := []string{"virtualenv venv", "source venv/bin/activate", "pip install -r requirements.txt", "ansible-galaxy install --roles-path roles -r requirements.yml", "cp swarm.ini.in swarm.ini"}
	setupCmd := exec.Command("/bin/sh", "-c", strings.Join(commands, "; "))
	setupCmd.Dir = path
	setupCmd.Stdout = os.Stdout
	setupCmd.Stderr = os.Stderr

	if err := setupCmd.Run(); err != nil {
		fmt.Printf("Error setting up virtualenv and installing dependencies for swarm cli: %s", err)
		os.Exit(1)
	}
}

func generatePemKeys(app string) {
	cmd := exec.Command("/bin/sh", "-c", "echo \"server.pem\" | ssh-keygen")
	cmd.Dir = filepath.Join(homeDir, ".send", app)

	if err := cmd.Run(); err != nil {
		fmt.Printf("Error generating server keys for %s: %s", app, err)
		os.Exit(1)
	}
}

func constructBundle(app string, ip string) {
	bundleDir := filepath.Join(homeDir, ".send", app)

	rootDir := getDirectory("starter")
	for _, file := range rootDir {
		downloadFile(file, bundleDir)
	}

	os.Mkdir(filepath.Join(bundleDir, "docker-compose"), os.ModePerm)
	dockerCompose := getDirectory("starter/docker-compose")
	for _, file := range dockerCompose {
		downloadFile(file, filepath.Join(bundleDir, "docker-compose"))
	}

	hostsData := []byte(fmt.Sprintf("[manager]\n%s", ip))
	err := ioutil.WriteFile(filepath.Join(bundleDir, "hosts"), hostsData, 0644)
	if err != nil {
		fmt.Printf("Error writing hosts file for %s: %s", app, err)
		os.Exit(1)
	}
}

func runSwarmOnServer(app string) {
	bundleDir := filepath.Join(homeDir, ".send", app)

	commands := []string{"python manage.py compile " + bundleDir, "python manage.py swarm lockdown", "python manage.py swarm join", "python manage.py swarm configure"}

	for _, command := range commands {
		fmt.Printf("RUNNING SWARM COMMAND: %s\n", command)
		cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("source venv/bin/activate; %s", command))
		cmd.Dir = filepath.Join(homeDir, ".send", "swarm-cli")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stdout

		if err := cmd.Run(); err != nil {
			fmt.Printf("Error running swarm cli command %s: %s", command, err)
			os.Exit(1)
		}
	}
}
