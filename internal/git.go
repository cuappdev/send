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

	"github.com/tidwall/gjson"
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

type fileRequest struct {
	Message string `json:"message"`
	Content string `json:"content"`
	Branch  string `json:"branch"`
	SHA     string `json:"sha"`
}

type referenceRequest struct {
	SHA string `json:"sha"`
}

type treeRequest struct {
	Tree     []tree `json:"tree"`
	BaseTree string `json:"base_tree"`
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
