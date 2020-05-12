package internal

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/digitalocean/godo"
)

var client = godo.NewFromToken(os.Getenv("DO_ACCESS_TOKEN"))

func createDroplet(name string, size string) int {
	createRequest := &godo.DropletCreateRequest{
		Name:   name,
		Region: "nyc3",
		Size:   size,
		Image: godo.DropletCreateImage{
			Slug: "ubuntu-18-04-x64",
		},
		SSHKeys: []godo.DropletCreateSSHKey{godo.DropletCreateSSHKey{
			Fingerprint: addSSHKey(name),
		}},
	}

	ctx := context.TODO()

	newDroplet, _, err := client.Droplets.Create(ctx, createRequest)

	if err != nil {
		fmt.Printf("Error creating new droplet: %s\n\n", err)
		os.Exit(1)
	}

	return newDroplet.ID
}

func addSSHKey(name string) string {
	homeDir, _ := os.UserHomeDir()
	keyPath := filepath.Join(homeDir, ".send", name, "server.pem.pub")

	publicKey, _ := ioutil.ReadFile(keyPath)

	createRequest := &godo.KeyCreateRequest{
		Name:      name,
		PublicKey: string(publicKey),
	}

	newKey, _, err := client.Keys.Create(context.TODO(), createRequest)

	if err != nil {
		fmt.Printf("Error adding new SSH key for %s onto DigitalOcean\n", name)
		os.Exit(1)
	}

	return newKey.Fingerprint
}

func getDroplet(id int) *godo.Droplet {
	droplet, _, err := client.Droplets.Get(context.TODO(), id)

	if err != nil {
		fmt.Printf("Error fetching droplet with id %d: %s \n", id, err)
		os.Exit(1)
	}
	return droplet
}

func getDropletIP(id int) string {
	droplet := getDroplet(id)

	ip, _ := droplet.PublicIPv4()
	return ip
}

func getDropletStatus(id int) string {
	droplet := getDroplet(id)
	return droplet.Status
}

func getValidSizes() []godo.Size {
	var sizes []godo.Size

	dropletSizes, _, err := client.Sizes.List(context.TODO(), nil)

	if err != nil {
		fmt.Printf("Error fetching droplet sizes: %s \n", err)
	}

	for _, size := range dropletSizes {
		if contains(size.Regions, "nyc3") {
			sizes = append(sizes, size)
		}
	}

	return sizes
}

func IsDropletSizeValid(sizeSlug string) bool {
	for _, size := range getValidSizes() {
		if size.Slug == sizeSlug {
			return true
		}
	}

	return false
}

func GetValidSizeStrings() []string {
	var sizes []string

	for _, size := range getValidSizes() {
		str := fmt.Sprintf("%s \t Memory: %d, Vcpus: %d, Disk: %d", size.Slug, size.Memory, size.Vcpus, size.Disk)
		sizes = append(sizes, str)
	}

	return sizes
}
