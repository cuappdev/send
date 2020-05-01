package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/crypto/bcrypt"
)

const baseURL = "https://github.coecis.cornell.edu/api/v3/repos/cuappdev/send-devops/contents/"

type fileRequest struct {
	Message string `json:"message"`
	Content string `json:"content"`
	Branch  string `json:"branch"`
	SHA     string `json:"sha"`
}

type user struct {
	Username       string   `json:"username"`
	HashedPassword string   `json:"hashed_password"`
	Apps           []string `json:"apps"`
	IsAdmin        bool     `json:"is_admin"`
}

func performRequest(method string, url string, body []byte) ([]byte, int) {
	client := &http.Client{}
	var bodyBuffer io.Reader

	if body == nil {
		bodyBuffer = nil
	} else {
		bodyBuffer = bytes.NewBuffer(body)
	}

	req, _ := http.NewRequest(method, url, bodyBuffer)
	req.Header.Set("Authorization", "token "+GetInstallationToken())
	resp, err := client.Do(req)

	if err != nil {
		return nil, resp.StatusCode
	}

	defer resp.Body.Close()
	respBody, _ := ioutil.ReadAll(resp.Body)

	return respBody, resp.StatusCode
}

func getContents(path string) []byte {
	contentsLink := baseURL + path
	res, statusCode := performRequest("GET", contentsLink, nil)

	if statusCode != 200 {
		return nil
	}

	return res
}

func getFile(path string) map[string]interface{} {
	var jsonRes map[string]interface{}
	json.Unmarshal(getContents(path), &jsonRes)

	return jsonRes
}

func getDirectory(path string) []map[string]interface{} {
	var jsonRes []map[string]interface{}
	json.Unmarshal(getContents(path), &jsonRes)

	return jsonRes
}

func downloadFile(file map[string]interface{}, outDir string) bool {
	if file["download_url"] != nil {
		downloadLink := file["download_url"].(string)
		cmd := exec.Command(
			"curl",
			"--output",
			filepath.Join(outDir, file["name"].(string)),
			"-L",
			downloadLink,
		)

		_, err := cmd.Output()
		if err != nil {
			fmt.Printf("error occurred downloading the config to %s : %s\n", outDir, err)
			return false
		}
	}
	return false
}

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

	contentsLink := baseURL + gitPath
	performRequest("PUT", contentsLink, body)
}

func RegisterUser(username string, password string) {
	contentsLink := baseURL + "users/" + username + ".json"

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

func contains(list []string, element string) bool {
	for _, x := range list {
		if x == element {
			return true
		}
	}
	return false
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

	contentsLink := baseURL + "users/" + username + ".json"
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
		fmt.Sprintf("appdev@%s-backend.cornellappdev.com", app),
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
