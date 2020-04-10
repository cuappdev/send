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
}

type user struct {
	Username       string   `json:"username"`
	HashedPassword string   `json:"hashed_password"`
	Apps           []string `json:"apps"`
}

func performRequest(method string, url string, body []byte) []byte {
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
		return nil
	}

	defer resp.Body.Close()
	respBody, _ := ioutil.ReadAll(resp.Body)

	return respBody
}

func getFile(path string) []byte {
	contentsLink := baseURL + path
	return performRequest("GET", contentsLink, nil)
}

func GetAppConfiguration(app string, path string) bool {
	fileRes := getFile(app + "/docker-compose")
	if fileRes == nil {
		fmt.Println("error occurred fetching config for %s", app)
		return false
	}

	var jsonRes []map[string]interface{}
	json.Unmarshal(fileRes, &jsonRes)

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
				fmt.Println("error occurred downloading the config for %s : %s", app, err)
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
	}
	userJson, _ := json.MarshalIndent(newUser, "", "\t")

	requestBody := fileRequest{
		"Register user " + username,
		base64.StdEncoding.Strict().EncodeToString(userJson),
		"master",
	}

	b, _ := json.Marshal(requestBody)

	performRequest("PUT", contentsLink, b)
}

func VerifyUser(username string, password []byte) (user, bool) {
	fileRes := getFile("users/" + username + ".json")

	var jsonRes map[string]interface{}
	json.Unmarshal(fileRes, &jsonRes)

	fileContents, _ := base64.StdEncoding.Strict().DecodeString(jsonRes["content"].(string))

	user := user{}
	json.Unmarshal(fileContents, &user)

	err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), password)

	if err != nil {
		return user, false
	}

	return user, true
}
