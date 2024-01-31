package drone

import (
	"context"

	"github.com/drone/drone-go/drone"
	"github.com/rancher/ecm-distro-tools/cmd/release/config"
	"golang.org/x/oauth2"
)

func newClient(ctx context.Context, server string, config config.Drone) drone.Client {
	conf := new(oauth2.Config)
	var token string
	switch server {
	case RancherPrServer:
		token = config.RancherPR
	case RancherPubServer:
		token = config.RancherPublish
	case K3sPrServer:
		token = config.K3sPR
	case K3sPubServer:
		token = config.K3sPublish
	}
	httpClient := conf.Client(ctx, &oauth2.Token{AccessToken: token})
	return drone.NewClient(server, httpClient)
}
