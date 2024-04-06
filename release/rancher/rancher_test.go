package rancher

import (
	"testing"
)

func TestCheckIfImageExists(t *testing.T) {
	img := "rancher/fleet-agent"
	imgVersion := "v0.8.2"
	exists, err := checkIfImageExists(img, imgVersion)
	if err != nil {
		t.Error(err)
	}
	if !exists {
		t.Error("image " + img + ":" + imgVersion + " doesn't exists")
	}
}
