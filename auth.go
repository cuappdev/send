package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/dgrijalva/jwt-go"
)

type credentials struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}

func generateJWTToken() string {
	signBytes, err := ioutil.ReadFile(os.Getenv("GIT_PEM_KEY_PATH"))

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	signKey, err := jwt.ParseRSAPrivateKeyFromPEM(signBytes)

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Minute * 10).Unix(),
		"iss": os.Getenv("GIT_APP_ID"),
	})

	tokenString, err := token.SignedString(signKey)
	return tokenString
}

func requestInstallationToken() string {
	client := &http.Client{}
	var bodyBuffer io.Reader = nil

	req, _ := http.NewRequest("POST", "https://github.coecis.cornell.edu/api/v3/app/installations/1/access_tokens", bodyBuffer)
	req.Header.Set("Authorization", "Bearer "+generateJWTToken())
	req.Header.Set("Accept", "application/vnd.github.machine-man-preview+json")
	resp, _ := client.Do(req)

	defer resp.Body.Close()
	respBody, _ := ioutil.ReadAll(resp.Body)

	var jsonRes map[string]interface{}
	json.Unmarshal(respBody, &jsonRes)

	writeCredentials(jsonRes["token"].(string), time.Now().Add(time.Hour).Unix())
	return jsonRes["token"].(string)
}

func getCredentialsPath() string {
	dir, _ := os.UserHomeDir()
	return path.Join(dir, ".send")
}

func writeCredentials(token string, expires_at int64) {
	os.Mkdir(getCredentialsPath(), 0755)

	credentials := credentials{token, expires_at}

	file, _ := json.MarshalIndent(credentials, "", "\t")

	_ = ioutil.WriteFile(path.Join(getCredentialsPath(), "credentials.json"), file, 0644)
}

func GetInstallationToken() string {
	file, err := ioutil.ReadFile(path.Join(getCredentialsPath(), "credentials.json"))

	if err != nil {
		return requestInstallationToken()
	}

	credentials := credentials{}
	_ = json.Unmarshal([]byte(file), &credentials)

	if credentials.ExpiresAt < time.Now().Unix() {
		return requestInstallationToken()
	}

	return credentials.Token
}

func WriteUser(username string) {
	// Encrypt username
	signBytes := []byte(os.Getenv("ENCRYPTION_KEY"))
	c, _ := aes.NewCipher(signBytes)
	gcm, _ := cipher.NewGCM(c)
	nonce := make([]byte, gcm.NonceSize())
	io.ReadFull(rand.Reader, nonce)

	encryptedUsername := gcm.Seal(nonce, nonce, []byte(username), nil)
	ioutil.WriteFile(path.Join(getCredentialsPath(), "user"), encryptedUsername, 0644)
}

func GetCurrentUser() string {
	file, err := ioutil.ReadFile(path.Join(getCredentialsPath(), "user"))

	if err != nil {
		return ""
	}

	// Decrypt username
	signBytes := []byte(os.Getenv("ENCRYPTION_KEY"))
	c, _ := aes.NewCipher(signBytes)
	gcm, _ := cipher.NewGCM(c)
	nonceSize := gcm.NonceSize()
	nonce, file := file[:nonceSize], file[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, file, nil)

	return string(plaintext)
}

func ClearCurrentUser() {
	err := os.Remove(path.Join(getCredentialsPath(), "user"))

	if os.IsNotExist(err) {
		fmt.Println("No user is currently logged in")
	} else {
		fmt.Println("Successfully logged out")
	}
}
