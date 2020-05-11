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
	"strings"

	"github.com/tidwall/gjson"
	"golang.org/x/crypto/bcrypt"
)

const baseURL = "https://github.coecis.cornell.edu/api/v3/repos/cuappdev/send-devops/"
const contentURL = baseURL + "contents/"
const gitURL = baseURL + "git/"

type blobRequest struct {
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
}

type commitRequest struct {
	Message string   `json:"message"`
	Tree    string   `json:"tree"`
	Parents []string `json:"parents"`
}

type referenceRequest struct {
	SHA string `json:"sha"`
}

type treeRequest struct {
	Tree     []tree `json:"tree"`
	BaseTree string `json:"base_tree"`
}

type fileRequest struct {
	Message string `json:"message"`
	Content string `json:"content"`
	Branch  string `json:"branch"`
	SHA     string `json:"sha"`
}

type tree struct {
	Path string `json:"path"`
	Mode string `json:"mode"`
	Type string `json:"type"`
	SHA  string `json:"sha"`
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
	contentsLink := contentURL + path
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
			fmt.Printf("error occurred downloading the file %s to %s : %s\n", file["name"].(string), outDir, err)
			return false
		}
		return true
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

func CommitBundle(app string) {
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

func createBlobs(app string, files *[]tree) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		data, err := ioutil.ReadFile(path)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		requestBody := blobRequest{
			base64.StdEncoding.Strict().EncodeToString(data),
			"base64",
		}
		body, _ := json.Marshal(requestBody)

		res, statusCode := performRequest("POST", gitURL+"blobs", body)

		if statusCode != 201 {
			fmt.Println("error trying to create git blob")
			os.Exit(1)
		}

		*files = append(*files, tree{
			path[strings.LastIndex(path, app):],
			"100644",
			"blob",
			gjson.GetBytes(res, "sha").String(),
		})
		return nil
	}
}

func createTree(files []tree) string {
	treeBody, _ := json.Marshal(treeRequest{
		files,
		getMasterSHA(),
	})
	treeRes, statusCode := performRequest("POST", gitURL+"trees", treeBody)
	if statusCode != 201 {
		fmt.Println("error trying to create git tree")
		os.Exit(1)
	}

	return gjson.GetBytes(treeRes, "sha").String()
}

func createCommit(app string, treeSHA string) string {
	commitBody, _ := json.Marshal(commitRequest{
		"Add deployment bundle for new app: " + app,
		treeSHA,
		[]string{getMasterSHA()},
	})
	commitRes, statusCode := performRequest("POST", gitURL+"commits", commitBody)
	if statusCode != 201 {
		fmt.Println("error trying to create git commit")
		os.Exit(1)
	}

	return gjson.GetBytes(commitRes, "sha").String()
}

func getMasterSHA() string {
	res, statusCode := performRequest("GET", baseURL+"branches/master", nil)

	if statusCode != 200 {
		fmt.Println("error fetching SHA of master")
		os.Exit(1)
	}

	return gjson.GetBytes(res, "commit.sha").String()
}
