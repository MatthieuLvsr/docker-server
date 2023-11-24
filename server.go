package main

import (
	// "fmt"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	// "os"
	// "os/exec"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/template/html/v2"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
)

type Tag struct {
	Id     int    `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type Tags struct {
	Count    int    `json:"count"`
	Next     int    `json:"next"`
	Previous string `json:"previous"`
	Results  []Tag  `json:"results"`
}

func main() {
	f, err := os.OpenFile(fmt.Sprintf("./logs/%s.log", time.Now().Format("2006-01-02T15-04-05-0700")), os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(f)

	log.Info("Server start...")

	engine := html.New("./public", ".html")

	app := fiber.New(fiber.Config{
		Views: engine,
	})

	lastCount := 1

	// go func(){
	c := cron.New()
	c.AddFunc("@every 10s", func() {

		result := getResponse()
		tags := parseResponse(result)

		if lastCount != tags.Count {
			lastCount = tags.Count
			log.Info("New version detected")
			update()
			log.Infof("New version pulled : %s", tags.Results[1].Name)
		}
	})
	c.Start()
	// }();

	log.Fatal(app.Listen(":3000"))
}

func getResponse() string {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	uri := os.Getenv("DOCKER_HUB_URI")
	if uri == "" {
		log.Fatal(uri)
	}
	resp, err := httpClient.Get(uri)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	return string(body)
}

func parseResponse(body string) Tags {
	var result Tags
	err := json.Unmarshal([]byte(body), &result)
	if err != nil {
		log.Fatal(err)
	}
	return result
}

func update() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	image := os.Getenv("IMAGE_NAME")
	if image == "" {
		log.Fatal(image)
	}
	name := strings.Split(image, "/")[1]

	cmd := exec.Command("docker", "stop", name)
	log.Info("Stoping last image...")
	err = cmd.Run()
	if err != nil {
		log.Error("Failed to stop:", err)
	} else {
		log.Info("Stoped successfully!")
	}

	cmd = exec.Command("docker", "rm", "-f", name)
	log.Info("Deleting old container...")
	err = cmd.Run()
	if err != nil {
		log.Error("Failed to delete:", err)
	} else {
		log.Info("Deleted successfully!")
	}

	cmd = exec.Command("docker", "rmi", "-f", image)
	log.Info("Deleting old image...")
	err = cmd.Run()
	if err != nil {
		log.Error("Failed to delete:", err)
	} else {
		log.Info("Deleted successfully!")
	}

	imageVersion := fmt.Sprintf("%s:latest", image)
	cmd = exec.Command("docker", "pull", imageVersion)
	log.Info("Checking for updates...")
	err = cmd.Run()
	if err != nil {
		log.Error("Failed to update:", err)
	} else {
		log.Info("Updated successfully!")
	}
	go func(name string, image string) {
		cmd = exec.Command("docker", "run", "-p", "80:80", "--name", name, image)
		log.Info("New image running...")
		err = cmd.Run()
	}(name, image)
}
