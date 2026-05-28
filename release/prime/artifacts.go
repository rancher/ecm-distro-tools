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
	scansBaseURL            = "https://scans.rancher.com"
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
	Rancher      ArtifactsIndexVersions
	RKE2         ArtifactsIndexVersions
	K3s          ArtifactsIndexVersions
	BaseURL      string
	ScansBaseURL string
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
	if err := os.WriteFile(filepath.Join(outPath, "index.html"), gaIndex, 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outPath, "index-prerelease.html"), preReleaseIndex, 0o644)
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
			BaseURL:      rancherArtifactsBaseURL,
			ScansBaseURL: scansBaseURL,
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
			BaseURL:      rancherArtifactsBaseURL,
			ScansBaseURL: scansBaseURL,
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

// stripBuildMeta removes semver build metadata (the "+..." suffix) from a version string.
// For example, "v1.30.1+rke2r1" becomes "v1.30.1".
func stripBuildMeta(version string) string {
	if i := strings.IndexByte(version, '+'); i >= 0 {
		return version[:i]
	}
	return version
}

func generateArtifactsHTML(content ArtifactsIndexContentGroup) ([]byte, error) {
	funcMap := template.FuncMap{
		"stripBuildMeta": stripBuildMeta,
	}
	tmpl, err := template.New("release-artifacts-index").Funcs(funcMap).Parse(artifactsIndexTemplate)
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
		<link rel="preconnect" href="https://fonts.googleapis.com">
		<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
		<link href="https://fonts.googleapis.com/css2?family=Lato:wght@400;700&family=Poppins:wght@300;500&display=swap" rel="stylesheet">
		<style>
			:root {
				--primary:        #2F68DF;
				--primary-hover:  #1F58CF;
				--primary-focus:  rgba(47, 104, 223, 0.15);
				--primary-text:   #FFFFFF;
				--body-bg:        #FFFFFF;
				--body-text:      #141419;
				--muted:          #6C6C76;
				--border:         #DCDEE7;
				--box-bg:         #F4F5FA;
				--link:           #3458A8;
				--header-height:  55px;
				--border-radius:  4px;
				--max-width:      1440px;
				/* search icon: magnifying glass (stroke #6C6C76, matches --muted) */
				--search-icon: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='16' height='16' viewBox='0 0 24 24' fill='none' stroke='%236C6C76' stroke-width='2'%3E%3Ccircle cx='11' cy='11' r='8'/%3E%3Cpath d='m21 21-4.35-4.35'/%3E%3C/svg%3E");
			}

			*, *::before, *::after { box-sizing: border-box; }

			body {
				margin: 0;
				font-family: 'Lato', Arial, Helvetica, sans-serif;
				font-size: 14px;
				line-height: 1.6;
				background-color: var(--body-bg);
				color: var(--body-text);
			}

			a {
				color: var(--link);
				text-decoration: none;
			}

			a:hover { text-decoration: underline; }

			header {
				display: flex;
				flex-direction: row;
				align-items: center;
				height: var(--header-height);
				padding: 0 24px;
				background-color: var(--body-bg);
				border-bottom: 1px solid var(--border);
				gap: 16px;
				position: sticky;
				top: 0;
				z-index: 100;
				box-shadow: 0 1px 4px rgba(0,0,0,0.06);
			}

			#rancher-logo { width: 180px; }

			header h1 {
				margin: 0;
				font-family: 'Poppins', sans-serif;
				font-size: 18px;
				font-weight: 500;
				color: var(--body-text);
				letter-spacing: 0.02em;
			}

			main {
				max-width: var(--max-width);
				margin: 0 auto;
				padding: 24px;
			}

			.project {
				margin-bottom: 32px;
				border: 1px solid var(--border);
				border-radius: var(--border-radius);
				overflow: hidden;
			}

			.project-header {
				display: flex;
				flex-direction: row;
				align-items: center;
				gap: 12px;
				padding: 12px 16px;
				background-color: var(--box-bg);
				border-bottom: 1px solid var(--border);
			}

			.project-header h2 {
				margin: 0;
				font-family: 'Poppins', sans-serif;
				font-size: 16px;
				font-weight: 500;
				color: var(--body-text);
			}

			.project-releases { padding: 8px 0; }

			.release {
				padding: 8px 16px 8px 24px;
				border-bottom: 1px solid var(--border);
			}

			.release:last-child { border-bottom: none; }

			.flex-row {
				display: flex;
				flex-direction: row;
				align-items: center;
				gap: 12px;
			}

			.release-title-tag {
				font-family: 'Lato', sans-serif;
				font-size: 14px;
				font-weight: 700;
				color: var(--body-text);
				min-width: 160px;
			}

			.btn {
				display: inline-flex;
				align-items: center;
				justify-content: center;
				height: 30px;
				padding: 0 12px;
				font-family: 'Lato', sans-serif;
				font-size: 12px;
				font-weight: 700;
				border-radius: var(--border-radius);
				border: 1px solid transparent;
				cursor: pointer;
				transition: background-color 0.1s ease, color 0.1s ease, border-color 0.1s ease;
				white-space: nowrap;
			}

			.btn-primary {
				background-color: var(--primary);
				color: var(--primary-text);
				border-color: var(--primary);
			}

			.btn-primary:hover {
				background-color: var(--primary-hover);
				border-color: var(--primary-hover);
			}

			.btn-secondary {
				background-color: var(--body-bg);
				color: var(--primary);
				border-color: var(--primary);
			}

			.btn-secondary:hover {
				background-color: var(--box-bg);
			}

			.files {
				padding: 8px 0 4px 16px;
			}

			.files ul {
				margin: 4px 0;
				padding-left: 16px;
				list-style: disc;
			}

			.files li {
				margin: 3px 0;
				font-size: 13px;
				color: var(--muted);
				word-break: break-all;
			}

			.files li a {
				color: var(--link);
			}

			.hidden { display: none; }

			.anchor {
				opacity: 0;
				margin-right: 6px;
				text-decoration: none;
				color: var(--muted);
				font-size: 16px;
				line-height: 1;
			}

			.flex-row:hover .anchor,
			.project-header:hover .anchor,
			.anchor:focus { opacity: 1; }

			.btn:focus-visible {
				outline: 2px solid var(--primary);
				outline-offset: 2px;
			}

			.search-wrapper {
				margin-bottom: 24px;
			}

			.search-input {
				width: 100%;
				max-width: 400px;
				height: 34px;
				padding: 0 12px 0 36px;
				font-family: 'Lato', sans-serif;
				font-size: 14px;
				color: var(--body-text);
				background-color: var(--body-bg);
				border: 1px solid var(--border);
				border-radius: var(--border-radius);
				background-image: var(--search-icon);
				background-repeat: no-repeat;
				background-position: 10px center;
				outline: none;
				transition: border-color 0.1s ease, box-shadow 0.1s ease;
			}

			.search-input:focus {
				border-color: var(--primary);
				box-shadow: 0 0 0 2px var(--primary-focus);
			}

			.badge {
				display: inline-flex;
				align-items: center;
				height: 20px;
				padding: 0 8px;
				font-size: 11px;
				font-weight: 700;
				color: var(--muted);
				background-color: var(--box-bg);
				border: 1px solid var(--border);
				border-radius: 10px;
				white-space: nowrap;
			}

			.no-results {
				padding: 16px 24px;
				color: var(--muted);
				font-size: 13px;
				font-style: italic;
				display: none;
			}

			.scan-link {
				display: inline-flex;
				align-items: center;
				gap: 4px;
				height: 30px;
				padding: 0 10px;
				font-family: 'Lato', sans-serif;
				font-size: 12px;
				font-weight: 700;
				color: var(--muted);
				border: 1px solid var(--border);
				border-radius: var(--border-radius);
				text-decoration: none;
				transition: color 0.1s ease, border-color 0.1s ease;
				white-space: nowrap;
			}

			.scan-link:hover {
				color: var(--primary);
				border-color: var(--primary);
				text-decoration: none;
			}
		</style>
	</head>
	<body>
		<header>
			<img src="https://prime.ribs.rancher.io/assets/img/rancher-suse-logo-horizontal-color.svg" alt="Rancher logo" id="rancher-logo" />
			<h1>Prime Artifacts</h1>
		</header>
		<main>
			<div class="search-wrapper">
				<input type="search" id="version-search" class="search-input"
					placeholder="Filter versions…"
					oninput="filterVersions(this.value)"
					autocomplete="off" />
			</div>
			<div class="project-rancher project">
				<div class="project-header">
					<a class="anchor" href="#rancher">#</a>
					<h2 id="rancher">rancher</h2>
					<button onclick="toggleProject('rancher')" id="project-rancher-expand" class="btn btn-secondary">hide</button>
				</div>
				<div id="project-rancher-releases" class="project-releases">
					<p class="no-results">No versions match your filter.</p>
					{{ range $i, $version := .Rancher.Versions }}
					<div id="rancher-{{ $version }}" class="release">
						<div class="flex-row">
							<a class="anchor" href="#rancher-{{ $version }}">#</a>
							<b class="release-title-tag">{{ $version }}</b>
							<span class="badge">{{ len (index $.Rancher.VersionsFiles $version) }} files</span>
							<a class="scan-link" href="{{ $.ScansBaseURL }}/rancher-{{ $version | stripBuildMeta }}.html" target="_blank" rel="noopener">CVE scan ↗</a>
							<button onclick="toggleFiles('{{ $version }}')" id="release-{{ $version }}-expand" class="btn btn-primary">show</button>
						</div>
						<div class="files hidden" id="release-{{ $version }}-files">
							<ul>{{ range index $.Rancher.VersionsFiles $version }}
							<li><a href="{{ $.BaseURL }}/rancher/{{ $version | urlquery }}/{{ . }}">{{ $.BaseURL }}/rancher/{{ $version }}/{{ . }}</a></li>
							{{ end }}</ul>
						</div>
					</div>{{ end }}
				</div>
			</div>
			<div class="project-rke2 project">
				<div class="project-header">
					<a class="anchor" href="#rke2">#</a>
					<h2 id="rke2">rke2</h2>
					<button onclick="toggleProject('rke2')" id="project-rke2-expand" class="btn btn-secondary">hide</button>
				</div>
				<div id="project-rke2-releases" class="project-releases">
					<p class="no-results">No versions match your filter.</p>
					{{ range $i, $version := .RKE2.Versions }}
					<div id="rke2-{{ $version }}" class="release">
						<div class="flex-row">
							<a class="anchor" href="#rke2-{{ $version }}">#</a>
							<b class="release-title-tag">{{ $version }}</b>
							<span class="badge">{{ len (index $.RKE2.VersionsFiles $version) }} files</span>
							<a class="scan-link" href="{{ $.ScansBaseURL }}/rke2-{{ $version | stripBuildMeta }}.html" target="_blank" rel="noopener">CVE scan ↗</a>
							<button onclick="toggleFiles('{{ $version }}')" id="release-{{ $version }}-expand" class="btn btn-primary">show</button>
						</div>
						<div class="files hidden" id="release-{{ $version }}-files">
							<ul>
							{{ range index $.RKE2.VersionsFiles $version }}
							<li><a href="{{ $.BaseURL }}/rke2/{{ $version | urlquery }}/{{ . }}">{{ $.BaseURL }}/rke2/{{ $version }}/{{ . }}</a></li>
							{{ end }}
							</ul>
						</div>
					</div>
					{{ end }}
				</div>
			</div>
			<div class="project-k3s project">
				<div class="project-header">
					<a class="anchor" href="#k3s">#</a>
					<h2 id="k3s">k3s</h2>
					<button onclick="toggleProject('k3s')" id="project-k3s-expand" class="btn btn-secondary">hide</button>
				</div>
				<div id="project-k3s-releases" class="project-releases">
					<p class="no-results">No versions match your filter.</p>
					{{ range $i, $version := .K3s.Versions }}
					<div id="k3s-{{ $version }}" class="release">
						<div class="flex-row">
							<a class="anchor" href="#k3s-{{ $version }}">#</a>
							<b class="release-title-tag">{{ $version }}</b>
							<span class="badge">{{ len (index $.K3s.VersionsFiles $version) }} files</span>
							<a class="scan-link" href="{{ $.ScansBaseURL }}/k3s-{{ $version | stripBuildMeta }}.html" target="_blank" rel="noopener">CVE scan ↗</a>
							<button onclick="toggleFiles('{{ $version }}')" id="release-{{ $version }}-expand" class="btn btn-primary">show</button>
						</div>
						<div class="files hidden" id="release-{{ $version }}-files">
							<ul>
							{{ range index $.K3s.VersionsFiles $version }}
							<li><a href="{{ $.BaseURL }}/k3s/{{ $version | urlquery }}/{{ . }}">{{ $.BaseURL }}/k3s/{{ $version }}/{{ . }}</a></li>
							{{ end }}
							</ul>
						</div>
					</div>
					{{ end }}
				</div>
			</div>
		</main>
		<script>
		function toggleProject(project) {
			const projectId = "project-" + project + "-releases"
			const expandButtonId = "project-" + project + "-expand"
			toggleSection(projectId, expandButtonId)
		}
		function toggleFiles(tag) {
			const filesId = "release-" + tag + "-files"
			const expandButtonId = "release-" + tag + "-expand"
			toggleSection(filesId, expandButtonId)
		}
		function toggleSection(sectionId, buttonId) {
			const button = document.getElementById(buttonId)
			document.getElementById(sectionId).classList.toggle("hidden")
			if (button.innerText === "hide") {
				button.innerText = "show"
				button.classList.replace("btn-secondary", "btn-primary")
			} else {
				button.innerText = "hide"
				button.classList.replace("btn-primary", "btn-secondary")
			}
		}
		function filterVersions(query) {
			const q = query.trim().toLowerCase()
			document.querySelectorAll('.project').forEach(function(project) {
				let visibleCount = 0
				project.querySelectorAll('.release').forEach(function(release) {
					const tag = release.querySelector('.release-title-tag')
					const match = !q || (tag && tag.textContent.trim().toLowerCase().includes(q))
					release.style.display = match ? '' : 'none'
					if (match) visibleCount++
				})
				const noResults = project.querySelector('.no-results')
				if (noResults) noResults.style.display = visibleCount === 0 ? '' : 'none'
			})
		}
		document.addEventListener('DOMContentLoaded', function() {
			const hash = window.location.hash.slice(1)
			if (!hash) return
			const releaseEl = document.getElementById(hash)
			if (!releaseEl || !releaseEl.classList.contains('release')) return
			const filesEl = releaseEl.querySelector('.files')
			const btnEl = releaseEl.querySelector('.btn')
			if (filesEl && btnEl) {
				filesEl.classList.remove('hidden')
				btnEl.innerText = 'hide'
				btnEl.classList.replace('btn-primary', 'btn-secondary')
				setTimeout(function() {
					releaseEl.scrollIntoView({ behavior: 'smooth', block: 'start' })
				}, 100) // small delay lets the browser paint the expanded content before scrolling
			}
		})
		</script>
	</body>
</html>
{{end}}`
