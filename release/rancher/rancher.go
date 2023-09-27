package rancher

import (
	"bufio"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	rancherImagesBaseURL  = "https://github.com/rancher/rancher/releases/download/"
	rancherImagesFileName = "/rancher-images.txt"
)

func ListRancherImagesRC(tag string) (string, error) {
	downloadURL := rancherImagesBaseURL + tag + rancherImagesFileName
	imagesFile, err := rancherImages(downloadURL)
	if err != nil {
		return "", err
	}
	rcImages := nonMirroredRCImages(imagesFile)

	if len(rcImages) == 0 {
		return "There are none non-mirrored images still in rc form for tag " + tag, nil
	}

	output := "The following non-mirrored images for tag *" + tag + "* are still in RC form\n```\n"
	for _, image := range rcImages {
		output += image + "\n"
	}
	output += "```"

	return output, nil
}

func nonMirroredRCImages(images string) []string {
	var rcImages []string

	scanner := bufio.NewScanner(strings.NewReader(images))
	for scanner.Scan() {
		image := scanner.Text()
		if strings.Contains(image, "mirrored") {
			continue
		}
		if strings.Contains(image, "-rc") {
			rcImages = append(rcImages, image)
		}
	}
	return rcImages
}

func rancherImages(imagesURL string) (string, error) {
	httpClient := http.Client{Timeout: time.Second * 15}
	logrus.Debug("downloading: " + imagesURL)
	resp, err := httpClient.Get(imagesURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", errors.New("failed to download rancher-images.txt file, expected status code 200, got: " + strconv.Itoa(resp.StatusCode))
	}
	images, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(images), nil
}
