package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/drone/drone-go/drone"
	"github.com/google/go-github/v80/github"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"

	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	"golang.org/x/oauth2"
)

var (
	version string
	gitSHA  string
)

var (
	vers      bool
	repo      string
	output    string
	milestone string
	branch    string
)

const usage = `version: %s
Usage: %[2]s [-r repo] [-m milestone] [-b branch] [-o output]
Options:
    -h               help
    -v               show version and exit
    -o output        output format (trace)
    -r repo          repository (rke2)
    -m milestone     milestone
    -b branch        release branch
Examples:
    %[2]s -r rke2 -m v1.21.5+rke2r1 -b release-1.21 -o trace > trace.json
`

func main() {
	flag.Usage = func() {
		w := os.Stderr
		if slices.Contains(os.Args, "-h") {
			w = os.Stdout
		}
		if _, err := fmt.Fprintf(w, usage, version, os.Args[0]); err != nil {
			logrus.Fatal(err.Error())
		}
	}

	flag.BoolVar(&vers, "v", false, "")
	flag.StringVar(&repo, "r", "rke2", "")
	flag.StringVar(&milestone, "m", "", "milestone")
	flag.StringVar(&branch, "b", "", "release branch")
	flag.StringVar(&output, "o", "trace", "output format (trace)")
	flag.Parse()

	if vers {
		logrus.Infof("version: %s - git sha: %s\n", version, gitSHA)
		os.Exit(0)
	}
	if repo != "rke2" {
		logrus.Fatalln("error: only supported repo is rke2")
	}
	if output != "trace" {
		logrus.Fatalln("error: only supported output format is trace")
	}

	ghToken := os.Getenv("GITHUB_TOKEN")
	if ghToken == "" {
		logrus.Fatalln("error: github token required")
	}

	dronePubToken := os.Getenv("DRONE_PUB_TOKEN")
	if dronePubToken == "" {
		logrus.Fatalln("error: drone-publish.rancher.io token required")
	}

	dronePrToken := os.Getenv("DRONE_PR_TOKEN")
	if dronePrToken == "" {
		logrus.Fatalln("error: drone-pr.rancher.io token required")
	}

	ctx := context.Background()
	w := io.Writer(os.Stdout)
	exp, err := makeGrafanaJSONTraceExporter(w)
	if err != nil {
		logrus.Fatalf("failed to crete grafana json trace exporter %v", err)
	}

	tp := trace.NewTracerProvider(
		trace.WithSampler(trace.AlwaysSample()),
		trace.WithBatcher(exp),
		trace.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceInstanceIDKey.String("1"),
				semconv.ServiceNamespaceKey.String("rancher"),
				semconv.ServiceNameKey.String("ecm-distro-tools"),
				semconv.ServiceVersionKey.String(version),
			),
		),
	)
	defer tp.Shutdown(context.Background())
	otel.SetTracerProvider(tp)

	ghConfig := new(oauth2.Config)
	gh := github.NewClient(ghConfig.Client(ctx, &oauth2.Token{AccessToken: ghToken}))

	dronePubConf := new(oauth2.Config)
	dronePub := dronePubConf.Client(ctx, &oauth2.Token{AccessToken: dronePubToken})

	dronePrConf := new(oauth2.Config)
	dronePr := dronePrConf.Client(ctx, &oauth2.Token{AccessToken: dronePrToken})

	client := Client{
		PullRequests: gh.PullRequests,
		Repositories: gh.Repositories,
		DronePr:      drone.NewClient("https://drone-pr.rancher.io", dronePr),
		DronePub:     drone.NewClient("https://drone-publish.rancher.io", dronePub),
	}
	rke2, err := client.getRKE2Release(ctx, milestone, branch)
	if err != nil {
		logrus.Fatalf("failed to get RKE2 release %v", err)
	}

	_, _, err = rke2.trace()
	if err != nil {
		logrus.Fatalf("failed to trace RKE2 release %v", err)
	}
}
