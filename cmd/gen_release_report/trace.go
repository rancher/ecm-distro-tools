package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v80/github"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func (r *RKE2Release) trace() (context.Context, trace.Span, error) {
	// create root span starting at the kubernetes/kubernetes release and ending at the RKE2 GA release
	traceOpts := []trace.SpanStartOption{
		trace.WithTimestamp(r.k8s.PublishedAt.Time),
		trace.WithAttributes(attribute.String("service", "github")),
		trace.WithAttributes(attribute.String("repo", "rancher/rke2")),
		trace.WithAttributes(attribute.String("event", "pull_request")),
		trace.WithAttributes(attribute.String("ref", "/refs/tags/"+*r.ga.TagName)),
		trace.WithAttributes(attribute.String("tag", r.version)),
		trace.WithAttributes(attribute.String("link", *r.ga.HTMLURL)),
	}

	k8sTracer := otel.Tracer("github")
	k8sCtx, k8sSpan := k8sTracer.Start(context.Background(), r.version, traceOpts...)
	k8sSpan.End(trace.WithTimestamp(r.ga.PublishedAt.Time))

	// trace builds triggered by rancher/image-build-kubernetes
	builds := k8sBuildsByRelease(r.builds, r.k8s)
	for _, build := range builds {
		build.trace(k8sCtx)
	}

	// trace builds triggered by the RKE2 GA release
	builds = rke2BuildsByRelease(r.builds, r.ga)
	for _, build := range builds {
		buildCtx := build.trace(k8sCtx)
		pkgBuilds := rke2PkgBuildsByBuild(build, r.builds)

		for _, pkgBuild := range pkgBuilds {
			pkgBuild.trace(buildCtx)
		}
	}

	// trace every release and every build triggered by the release
	for _, rc := range r.rcs {
		releaseEnd := rc.CreatedAt.Add(time.Minute)
		// start a span for the release
		traceOpts = []trace.SpanStartOption{
			trace.WithAttributes(attribute.String("service", "github")),
			trace.WithAttributes(attribute.String("event", "release")),
			trace.WithAttributes(attribute.String("repo", "rancher/rke2")),
			trace.WithAttributes(attribute.String("tag", *rc.TagName)),
			trace.WithTimestamp(rc.PublishedAt.Time),
		}
		releaseCtx, releaseSpan := otel.Tracer("github").Start(k8sCtx, *rc.TagName, traceOpts...)

		builds := rke2BuildsByRelease(r.builds, rc)
		for _, build := range builds {
			build.trace(releaseCtx)
			releaseEnd = latest(releaseEnd, time.Unix(build.Finished, 0))
		}

		releaseSpan.End(trace.WithTimestamp(releaseEnd))
	}

	// trace every pull request and every build triggered by the pull request
	for _, pr := range r.prs {
		pullEnd := *pr.pull.MergedAt.GetTime()
		// start a span for the release
		traceOpts = []trace.SpanStartOption{
			trace.WithAttributes(attribute.String("service", "github")),
			trace.WithAttributes(attribute.String("event", "pull_request")),
			trace.WithAttributes(attribute.String("repo", "rancher/rke2")),
			trace.WithAttributes(attribute.String("ref", *pr.pull.Base.Ref)),
			trace.WithAttributes(attribute.String("link", *pr.pull.HTMLURL)),
			trace.WithTimestamp(*pr.pull.CreatedAt.GetTime()),
		}
		tracer := otel.Tracer("github")
		name := fmt.Sprintf("#%d", *pr.pull.Number)
		pullCtx, pullSpan := tracer.Start(k8sCtx, name, traceOpts...)

		builds := rke2BuildsByPullRequest(r.builds, pr)
		for _, build := range builds {
			build.trace(pullCtx)
			pullEnd = latest(pullEnd, time.Unix(build.Finished, 0))
		}

		pullSpan.End(trace.WithTimestamp(pullEnd))
	}

	return k8sCtx, k8sSpan, nil
}

func (b *Build) trace(ctx context.Context) context.Context {
	startOpts := []trace.SpanStartOption{
		trace.WithTimestamp(time.Unix(b.Started, 0)),
		trace.WithAttributes(attribute.String("service", "drone")),
		trace.WithAttributes(attribute.Int64("drone_id", b.ID)),
		trace.WithAttributes(attribute.Int64("drone_number", b.Number)),
		trace.WithAttributes(attribute.String("drone_status", b.Status)),
		trace.WithAttributes(attribute.String("ref", b.Ref)),
		trace.WithAttributes(attribute.String("event", b.Event)),
		trace.WithAttributes(attribute.String("source", b.Source)),
		trace.WithAttributes(attribute.String("link", b.Link)),
		trace.WithAttributes(attribute.String("drone_action", b.Action)),
	}
	tracer := otel.GetTracerProvider().Tracer("drone")
	name := fmt.Sprintf("%s build %s/%s %d", b.Status, b.Owner, b.Repo, b.Number)
	buildCtx, buildSpan := tracer.Start(ctx, name, startOpts...)
	buildSpan.End(trace.WithTimestamp(time.Unix(b.Finished, 0)))
	return buildCtx
}

func rke2PkgBuildsByBuild(build *Build, pkgBuilds []*Build) []*Build {
	var result []*Build
	for _, pkg := range pkgBuilds {
		if build.Owner != "rancher" || build.Repo != "rke2-packaging" {
			continue
		}
		if strings.Contains(pkg.Ref, build.Ref) {
			result = append(result, pkg)
		}
	}
	return result
}

func rke2BuildsByPullRequest(builds []*Build, pr *Pull) []*Build {
	var result []*Build
	for _, build := range builds {
		if build.Owner != "rancher" || build.Repo != "rke2" {
			continue
		}
		if build.Event == "pull_request" {
			if build.Ref == fmt.Sprintf("refs/pull/%d/head", *pr.pull.Number) {
				result = append(result, build)
				continue
			}
		}
		if build.Event == "push" {
			if build.After == *pr.pull.MergeCommitSHA {
				result = append(result, build)
				continue
			}
			if pr.HasCommit(build.After) {
				result = append(result, build)
				break
			}
		}
	}
	return result
}

func rke2BuildsByRelease(builds []*Build, r *github.RepositoryRelease) []*Build {
	var result []*Build
	for _, build := range builds {
		if build.Owner != "rancher" || build.Repo != "rke2" {
			continue
		}
		if build.Event == "tag" && build.Ref == "refs/tags/"+*r.TagName {
			result = append(result, build)
		}
	}
	return result
}

func k8sBuildsByRelease(builds []*Build, r *github.RepositoryRelease) []*Build {
	var result []*Build
	for _, build := range builds {
		if build.Owner != "rancher" || build.Repo != "image-build-kubernetes" {
			continue
		}
		if build.Event == "tag" && strings.Contains(build.Ref, "refs/tags/"+*r.TagName) {
			result = append(result, build)
		}
	}
	return result
}

func (p *Pull) HasCommit(sha string) bool {
	for _, commit := range p.commits {
		if sha == *commit.SHA {
			return true
		}
	}
	return false
}
