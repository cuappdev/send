package internal

import (
	"fmt"
	"os"
	"os/exec"
)

func SendToSlack(message string) {
	messagePayload := fmt.Sprintf(`{"text":"%s"}`, message)
	slackHookURL := os.Getenv("SEND_UPDATES_HOOK_URL")
	cmd := exec.Command(
		"curl",
		"-X",
		"POST",
		"-H",
		"'Content-type: application/json'",
		"--data",
		messagePayload,
		slackHookURL,
	)
	_, err := cmd.Output()
	if err != nil {
		fmt.Printf("error occurred sending slack message: %s\n", err)
	}
}
