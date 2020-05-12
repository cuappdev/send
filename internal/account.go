package internal

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh/terminal"
)

func promptUsername() string {
	fmt.Print("Username: ")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	username := scanner.Text()
	if username == "" {
		fmt.Println("\nYour username cannot be empty. Try again.")
		os.Exit(1)
	}
	return username
}

func promptPassword(prompt string) []byte {
	fmt.Print(prompt)
	bytePassword, _ := terminal.ReadPassword(int(syscall.Stdin))

	if string(bytePassword) == "" {
		fmt.Println("\nYour password cannot be empty. Try again.")
		os.Exit(1)
	}
	return bytePassword
}

func Signup() (string, string) {
	username := promptUsername()

	bytePassword := promptPassword("Password: ")
	bytePassword2 := promptPassword("\nPassword again: ")

	if string(bytePassword) != string(bytePassword2) {
		fmt.Println("\nYou entered two different passwords. Try again.")
		os.Exit(1)
	}

	hash, _ := bcrypt.GenerateFromPassword(bytePassword, bcrypt.MinCost)

	return strings.TrimSpace(username), string(hash)
}

func Login() (string, []byte) {
	username := promptUsername()

	bytePassword := promptPassword("Password: ")

	return strings.TrimSpace(username), bytePassword
}
