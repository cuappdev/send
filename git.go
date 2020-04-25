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

func GetAppConfiguration(app string, path string) bool {
	jsonRes := getDirectory(app + "/docker-compose")
	if jsonRes == nil {
		fmt.Printf("error occurred fetching config for %s\n", app)
		return false
	}

	os.Mkdir(filepath.Join(path, app), os.ModePerm)

	for _, file := range jsonRes {
		if file["download_url"] != nil {
			downloadLink := file["download_url"].(string)
			cmd := exec.Command(
				"curl",
				"--output",
				filepath.Join(path, app, file["name"].(string)),
				"-L",
				downloadLink,
			)

			_, err := cmd.Output()
			if err != nil {
				fmt.Printf("error occurred downloading the config for %s : %s\n", app, err)
				return false
			}
		}
	}

	return true
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
