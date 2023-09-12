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
	imagesFile, err := getRancherImagesFile(tag)
	if err != nil {
		return "", err
	}
	rcImages, err := findRCNonMirroredImages(imagesFile)
	if err != nil {
		return "", err
	}

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

func findRCNonMirroredImages(images string) ([]string, error) {
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
	return rcImages, nil
}

func rancherImages(tag string) (string, error) {
	httpClient := http.Client{Timeout: time.Second * 15}
	downloadURL := rancherImagesBaseURL + tag + rancherImagesFileName
	logrus.Debug("downloading: " + downloadURL)
	resp, err := httpClient.Get(downloadURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", errors.New(
			"failed to download rancher-images.txt file, expected status code 200, got: " + strconv.Itoa(resp.StatusCode),
		)
	}

	images, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(images), nil
}
