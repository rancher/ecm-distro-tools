package rancher

import (
	"bufio"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

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

func findRCNonMirroredImages(imagesFile io.ReadCloser) ([]string, error) {
	var rcImages []string

	scanner := bufio.NewScanner(imagesFile)
	for scanner.Scan() {
		image := scanner.Text()
		if strings.Contains(image, "mirrored") {
			continue
		}
		if strings.Contains(image, "-rc") {
			rcImages = append(rcImages, image)
		}
	}
	return rcImages, imagesFile.Close()
}

func getRancherImagesFile(tag string) (io.ReadCloser, error) {
	downloadURL := rancherImagesBaseURL + tag + rancherImagesFileName
	logrus.Debug("downloading: " + downloadURL)
	resp, err := http.Get(downloadURL)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(
			"failed to download rancher-images.txt file, expected status code 200, got: " + strconv.Itoa(resp.StatusCode),
		)
	}

	return resp.Body, nil
}
