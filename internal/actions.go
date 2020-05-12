package internal

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func downloadPemKey(app string) {
	file := getFile(app + "/server.pem")
	if file == nil {
		fmt.Printf("error occurred pem key for %s\n", app)
		return
	}

	dir, _ := os.UserHomeDir()
	downloadDir := filepath.Join(dir, ".send", app)
	os.Mkdir(downloadDir, os.ModePerm)

	downloadFile(file, downloadDir)
	os.Chmod(filepath.Join(downloadDir, "server.pem"), 0600)
}

func GetAppConfiguration(app string, path string) bool {
	jsonRes := getDirectory(app + "/docker-compose")
	if jsonRes == nil {
		fmt.Printf("error occurred fetching config for %s\n", app)
		return false
	}

	os.Mkdir(filepath.Join(path, app), os.ModePerm)

	for _, file := range jsonRes {
		if !downloadFile(file, filepath.Join(path, app)) {
			return false
		}
	}

	return true
}

func PushAppConfiguration(username string, app string, path string) {
	fileName := filepath.Base(path)
	gitPath := app + "/docker-compose/" + fileName

	data, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	jsonRes := getFile(gitPath)
	var requestBody fileRequest
	if jsonRes != nil {
		requestBody = fileRequest{
			fmt.Sprintf("%s updated %s for %s", username, fileName, app),
			base64.StdEncoding.Strict().EncodeToString(data),
			"master",
			jsonRes["sha"].(string),
		}
	} else {
		requestBody = fileRequest{
			fmt.Sprintf("%s added %s for %s", username, fileName, app),
			base64.StdEncoding.Strict().EncodeToString(data),
			"master",
			"",
		}
	}

	body, _ := json.Marshal(requestBody)

	contentsLink := contentURL + gitPath
	_, statusCode := performRequest("PUT", contentsLink, body)

	if statusCode == 200 || statusCode == 201 {
		downloadPemKey(app)
		homeDir, _ := os.UserHomeDir()
		pemPath := filepath.Join(homeDir, ".send", app, "server.pem")

		cmd := exec.Command(
			"scp",
			"-i",
			pemPath,
			path,
			fmt.Sprintf("appdev@%s:docker-compose", getHost(app)),
		)

		_, err := cmd.Output()
		if err != nil {
			fmt.Printf("Error adding file %s onto %s: %s \n", path, app, err)
			os.Exit(1)
		}

		os.Remove(pemPath)
	}
}

func RegisterUser(username string, password string) {
	contentsLink := contentURL + "users/" + username + ".json"

	newUser := user{
		username,
		password,
		[]string{},
		false,
	}
	userJson, _ := json.MarshalIndent(newUser, "", "\t")

	requestBody := fileRequest{
		"Register user " + username,
		base64.StdEncoding.Strict().EncodeToString(userJson),
		"master",
		"",
	}

	b, _ := json.Marshal(requestBody)

	performRequest("PUT", contentsLink, b)
}

func getUserAndSHA(username string) (user, string) {
	fileRes := getFile("users/" + username + ".json")

	if fileRes == nil {
		fmt.Println("User does not exist.")
		os.Exit(1)
	}

	fileContents, _ := base64.StdEncoding.Strict().DecodeString(fileRes["content"].(string))

	user := user{}
	json.Unmarshal(fileContents, &user)

	return user, fileRes["sha"].(string)
}

func GetUser(username string) user {
	user, _ := getUserAndSHA(username)
	return user
}

func VerifyUser(username string, password []byte) (user, bool) {
	user := GetUser(username)
	err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), password)

	return user, err == nil
}

func AddApp(username string, app string) {
	user, sha := getUserAndSHA(username)

	if contains(user.Apps, app) {
		fmt.Printf("User %s already has access to %s\n", username, app)
		return
	}

	user.Apps = append(user.Apps, app)
	userJson, _ := json.MarshalIndent(user, "", "\t")

	requestBody := fileRequest{
		fmt.Sprintf("Grant app access to %s for %s", app, username),
		base64.StdEncoding.Strict().EncodeToString(userJson),
		"master",
		sha,
	}

	body, _ := json.Marshal(requestBody)

	contentsLink := contentURL + "users/" + username + ".json"
	performRequest("PUT", contentsLink, body)
}

func HasAccessTo(username string, app string) bool {
	user := GetUser(username)

	return contains(user.Apps, app)
}

func ExecCmd(app string, command string) string {
	downloadPemKey(app)
	homeDir, _ := os.UserHomeDir()
	pemPath := filepath.Join(homeDir, ".send", app, "server.pem")

	cmd := exec.Command(
		"ssh",
		"-i",
		pemPath,
		fmt.Sprintf("appdev@%s", getHost(app)),
		command,
	)

	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("error executing command for %s : %s\n", app, err)
		os.Exit(1)
	}

	os.Remove(pemPath)

	return string(output)
}

func getHost(app string) string {
	fileRes := getFile(app + "/hosts")

	if fileRes == nil {
		fmt.Println("Could not find specified app or hosts file")
		os.Exit(1)
	}

	fileContents, _ := base64.StdEncoding.Strict().DecodeString(fileRes["content"].(string))

	return strings.TrimSpace(strings.Split(string(fileContents), "\n")[1])
}

func commitBundle(app string) {
	homeDir, _ := os.UserHomeDir()
	dirPath := filepath.Join(homeDir, ".send", app)

	var files []tree
	filepath.Walk(dirPath, createBlobs(app, &files))

	treeSHA := createTree(files)
	commitSHA := createCommit(app, treeSHA)

	refBody, _ := json.Marshal(referenceRequest{commitSHA})
	_, statusCode := performRequest("PATCH", gitURL+"refs/heads/master", refBody)
	if statusCode != 200 {
		fmt.Println("error updating master with new commit")
		os.Exit(1)
	}
}
