package prime

import (
	"bytes"
	"context"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"golang.org/x/mod/semver"
)

const (
	rancherArtifactsBucket  = "prime-artifacts"
	rancherArtifactsPrefix  = "rancher/v"
	rke2ArtifactsPrefix     = "rke2/v"
	k3sArtifactsPrefix      = "k3s/v"
	rancherArtifactsBaseURL = "https://prime.ribs.rancher.io"
)

type ArtifactsIndexContent struct {
	GA         ArtifactsIndexContentGroup
	PreRelease ArtifactsIndexContentGroup
}

type ArtifactsIndexVersions struct {
	Versions      []string
	VersionsFiles map[string][]string
}

type ArtifactsIndexContentGroup struct {
	Rancher ArtifactsIndexVersions
	RKE2    ArtifactsIndexVersions
	K3s     ArtifactsIndexVersions
	BaseURL string
}

type ArtifactLister interface {
	List(ctx context.Context) (rancherKeys []string, rke2Keys []string, k3sKeys []string, err error)
}

type ArtifactBucket struct {
	bucket string
	client *s3.Client
}

func NewArtifactBucket(client *s3.Client) ArtifactBucket {
	return ArtifactBucket{
		bucket: rancherArtifactsBucket,
		client: client,
	}
}

func (a ArtifactBucket) List(ctx context.Context) ([]string, []string, []string, error) {
	rancherKeys, err := listS3Objects(ctx, a.client, a.bucket, rancherArtifactsPrefix)
	if err != nil {
		return nil, nil, nil, err
	}
	rke2Keys, err := listS3Objects(ctx, a.client, a.bucket, rke2ArtifactsPrefix)
	if err != nil {
		return nil, nil, nil, err
	}
	k3sKeys, err := listS3Objects(ctx, a.client, a.bucket, k3sArtifactsPrefix)
	if err != nil {
		return nil, nil, nil, err
	}
	return rancherKeys, rke2Keys, k3sKeys, nil
}

type ArtifactDir struct {
	dir string
}

func NewArtifactDir(dir string) ArtifactDir {
	return ArtifactDir{dir}
}

func (a ArtifactDir) List(ctx context.Context) ([]string, []string, []string, error) {
	var rancherKeys, rke2Keys, k3sKeys []string
	err := filepath.WalkDir(a.dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(a.dir, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, "rancher/v") {
			rancherKeys = append(rancherKeys, rel)
		} else if strings.HasPrefix(rel, "rke2/v") {
			rke2Keys = append(rke2Keys, rel)
		} else if strings.HasPrefix(rel, "k3s/v") {
			k3sKeys = append(k3sKeys, rel)
		}
		return nil
	})
	if err != nil {
		return nil, nil, nil, err
	}
	return rancherKeys, rke2Keys, k3sKeys, nil
}

// GenerateArtifactsIndex lists artifacts and writes index.html and index-prerelease.html
func GenerateArtifactsIndex(ctx context.Context, outPath string, ignoreVersions []string, lister ArtifactLister) error {
	ignore := make(map[string]bool, len(ignoreVersions))
	for _, v := range ignoreVersions {
		ignore[v] = true
	}
	rancherKeys, rke2Keys, k3sKeys, err := lister.List(ctx)
	if err != nil {
		return err
	}
	content := generateArtifactsIndexContent(rancherKeys, rke2Keys, k3sKeys, ignore)
	gaIndex, err := generateArtifactsHTML(content.GA)
	if err != nil {
		return err
	}
	preReleaseIndex, err := generateArtifactsHTML(content.PreRelease)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outPath, "index.html"), gaIndex, 0644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outPath, "index-prerelease.html"), preReleaseIndex, 0644)
}

func generateArtifactsIndexContent(rancherKeys, rke2Keys, k3sKeys []string, ignoreVersions map[string]bool) ArtifactsIndexContent {
	indexContent := ArtifactsIndexContent{
		GA: ArtifactsIndexContentGroup{
			Rancher: ArtifactsIndexVersions{
				Versions:      []string{},
				VersionsFiles: map[string][]string{},
			},
			RKE2: ArtifactsIndexVersions{
				Versions:      []string{},
				VersionsFiles: map[string][]string{},
			},
			K3s: ArtifactsIndexVersions{
				Versions:      []string{},
				VersionsFiles: map[string][]string{},
			},
			BaseURL: rancherArtifactsBaseURL,
		},
		PreRelease: ArtifactsIndexContentGroup{
			Rancher: ArtifactsIndexVersions{
				Versions:      []string{},
				VersionsFiles: map[string][]string{},
			},
			RKE2: ArtifactsIndexVersions{
				Versions:      []string{},
				VersionsFiles: map[string][]string{},
			},
			K3s: ArtifactsIndexVersions{
				Versions:      []string{},
				VersionsFiles: map[string][]string{},
			},
			BaseURL: rancherArtifactsBaseURL,
		},
	}

	indexContent.GA.Rancher, indexContent.PreRelease.Rancher = parseVersionsFromKeys(rancherKeys, "rancher/", ignoreVersions)
	indexContent.GA.RKE2, indexContent.PreRelease.RKE2 = parseVersionsFromKeys(rke2Keys, "rke2/", ignoreVersions)
	indexContent.GA.K3s, indexContent.PreRelease.K3s = parseVersionsFromKeys(k3sKeys, "k3s/", ignoreVersions)

	return indexContent
}

// parseVersionsFromKeys extracts versions and files from keys and returns GA and pre-release version structs
func parseVersionsFromKeys(keys []string, prefix string, ignoreVersions map[string]bool) (ArtifactsIndexVersions, ArtifactsIndexVersions) {
	var versions []string
	versionsFiles := make(map[string][]string)

	gaVersions := ArtifactsIndexVersions{
		Versions:      []string{},
		VersionsFiles: map[string][]string{},
	}

	preReleaseVersions := ArtifactsIndexVersions{
		Versions:      []string{},
		VersionsFiles: map[string][]string{},
	}

	for _, key := range keys {
		if !strings.Contains(key, prefix) {
			continue
		}
		keyFile := strings.Split(strings.TrimPrefix(key, prefix), "/")
		if len(keyFile) < 2 || keyFile[1] == "" {
			continue
		}
		version := keyFile[0]
		file := keyFile[1]

		if _, ok := ignoreVersions[version]; ok {
			continue
		}

		if _, ok := versionsFiles[version]; !ok {
			versions = append(versions, version)
		}
		versionsFiles[version] = append(versionsFiles[version], file)
	}

	semver.Sort(versions)

	// starting from the last index will result in a newest to oldest sorting
	for i := len(versions) - 1; i >= 0; i-- {
		version := versions[i]
		// only non ga releases contains '-' e.g: -rc, -hotfix
		if strings.Contains(version, "-") {
			preReleaseVersions.Versions = append(preReleaseVersions.Versions, version)
			preReleaseVersions.VersionsFiles[version] = versionsFiles[version]
		} else {
			gaVersions.Versions = append(gaVersions.Versions, version)
			gaVersions.VersionsFiles[version] = versionsFiles[version]
		}
	}

	return gaVersions, preReleaseVersions
}

func generateArtifactsHTML(content ArtifactsIndexContentGroup) ([]byte, error) {
	tmpl, err := template.New("release-artifacts-index").Parse(artifactsIndexTemplate)
	if err != nil {
		return nil, err
	}
	buff := bytes.NewBuffer(nil)
	if err := tmpl.ExecuteTemplate(buff, "release-artifacts-index", content); err != nil {
		return nil, err
	}

	return buff.Bytes(), nil
}

func listS3Objects(ctx context.Context, s3Client *s3.Client, bucketName string, prefix string) ([]string, error) {
	var keys []string
	var continuationToken *string
	isTruncated := true
	for isTruncated {
		objects, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            &bucketName,
			Prefix:            &prefix,
			ContinuationToken: continuationToken,
		})
		if err != nil {
			return nil, err
		}
		for _, object := range objects.Contents {
			keys = append(keys, *object.Key)
		}
		// used for pagination
		continuationToken = objects.NextContinuationToken
		// if the bucket has more keys
		if objects.IsTruncated != nil && !*objects.IsTruncated {
			isTruncated = false
		}
	}
	return keys, nil
}

const artifactsIndexTemplate = `{{ define "release-artifacts-index" }}
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta http-equiv="X-UA-Compatible" content="ie=edge">
    <title>Rancher Prime Artifacts</title>
    <link rel="icon" type="image/png" href="https://prime.ribs.rancher.io/assets/img/favicon.png">
    <style>
    body { font-family: 'Courier New', monospace, Verdana, Geneneva; }
    header { display: flex; flex-direction: row; justify-items: center; }
    #rancher-logo { width: 200px; }
    .project { margin-left: 20px; }
    .release { margin-left: 40px; margin-bottom: 20px; }
    .release h3 { margin-bottom: 0px; }
    .files { margin-left: 60px; display: flex; flex-direction: column; }
    .release-title { display: flex; flex-direction: row; }
    .release-title-tag { margin-right: 20px; min-width: 70px; }
    .release-title-expand { background-color: #2453ff; color: white; border-radius: 5px; border: none; }
    .release-title-expand:hover, .expand-active{ background-color: white; color: #2453ff; border: 1px solid #2453ff; }
    .hidden { display: none; overflow: hidden; }
	.anchor { opacity:0; margin-right:8px; text-decoration:none; color:dimgray; }
	.release-title:hover .anchor, h2:hover .anchor, .anchor:focus { opacity:1; }
    </style>
  </head>
  <body>
    <header>
      <img src="https://prime.ribs.rancher.io/assets/img/rancher-suse-logo-horizontal-color.svg" alt="rancher logo" id="rancher-logo" />
      <h1>PRIME ARTIFACTS</h1>
    </header>
    <main>
			<div class="project-rancher project">
				<h2 id="rancher">
				  <a class="anchor" href="#rancher">#</a>rancher
				</h2>
        {{ range $i, $version := .Rancher.Versions }}
        <div id="rancher-{{ $version }}" class="release-{{ $version }} release">
          <div class="release-title">
						<a class="anchor" href="#rancher-{{ $version }}">#</a>
						<b class="release-title-tag">{{ $version }}</b>
            <button onclick="expand('{{ $version }}')" id="release-{{ $version }}-expand" class="release-title-expand">expand</button>
          </div>
          <div class="files" id="release-{{ $version }}-files">
            <ul>
              {{ range index $.Rancher.VersionsFiles $version }}
              <li><a href="{{ $.BaseURL }}/rancher/{{ $version | urlquery }}/{{ . }}">{{ $.BaseURL }}/rancher/{{ $version }}/{{ . }}</a></li>
              {{ end }}
            </ul>
          </div>
        </div>
				{{ end }}
      </div>
	  <div class="project-rke2 project">
        <h2 id="rke2">
		  <a class="anchor" href="#rke2">#</a>rke2
		</h2>
        {{ range $i, $version := .RKE2.Versions }}
        <div id="rke2-{{ $version }}" class="release-{{ $version }} release">
          <div class="release-title">
		  				<a class="anchor" href="#rke2-{{ $version }}">#</a>
						<b class="release-title-tag">{{ $version }}</b>
            <button onclick="expand('{{ $version }}')" id="release-{{ $version }}-expand" class="release-title-expand">expand</button>
          </div>
          <div class="files" id="release-{{ $version }}-files">
            <ul>
              {{ range index $.RKE2.VersionsFiles $version }}
              <li><a href="{{ $.BaseURL }}/rke2/{{ $version | urlquery }}/{{ . }}">{{ $.BaseURL }}/rke2/{{ $version }}/{{ . }}</a></li>
              {{ end }}
            </ul>
          </div>
        </div>
		{{ end }}
      </div>
	  	  <div class="project-k3s project">
        <h2>k3s</h2>
        {{ range $i, $version := .K3s.Versions }}
        <div class="release-{{ $version }} release">
          <div class="release-title">
						<b class="release-title-tag">{{ $version }}</b>
            <button onclick="expand('{{ $version }}')" id="release-{{ $version }}-expand" class="release-title-expand">expand</button>
          </div>
          <div class="files" id="release-{{ $version }}-files">
            <ul>
              {{ range index $.K3s.VersionsFiles $version }}
              <li><a href="{{ $.BaseURL }}/k3s/{{ $version | urlquery }}/{{ . }}">{{ $.BaseURL }}/k3s/{{ $version }}/{{ . }}</a></li>
              {{ end }}
            </ul>
          </div>
        </div>
		{{ end }}
      </div>
    </main>
  <script>
    hideFiles()
    function expand(tag) {
      const filesId = "release-" + tag + "-files"
      const expandButtonId = "release-" + tag + "-expand"
      document.getElementById(filesId).classList.toggle("hidden")
      document.getElementById(expandButtonId).classList.toggle("expand-active")
    }
    function hideFiles() {
        const fileDivs = document.querySelectorAll(".files")
        fileDivs.forEach(f => f.classList.add("hidden"))
    }
  </script>
  </body>
</html>
{{end}}`
