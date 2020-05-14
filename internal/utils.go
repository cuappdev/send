package internal

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
)

func performRequest(method string, url string, body []byte) (responseBody []byte, statusCode int) {
	client := &http.Client{}
	var bodyBuffer io.Reader

	if body == nil {
		bodyBuffer = nil
	} else {
		bodyBuffer = bytes.NewBuffer(body)
	}

	req, _ := http.NewRequest(method, url, bodyBuffer)
	req.Header.Set("Authorization", "token "+getInstallationToken())
	resp, err := client.Do(req)

	if err != nil {
		return nil, resp.StatusCode
	}

	defer resp.Body.Close()
	respBody, _ := ioutil.ReadAll(resp.Body)

	return respBody, resp.StatusCode
}

func contains(list []string, element string) bool {
	for _, x := range list {
		if x == element {
			return true
		}
	}
	return false
}
