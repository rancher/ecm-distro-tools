package release

var (
	modifyScript = `
	#!/bin/bash
	cd %Workspace%
	git clone %K3SFork%
	cd %Workspace%/k3s
	git remote add upstream https://github.com/k3s-io/k3s.git
	git fetch upstream
	git branch delete {{ .NewK8SVersion }}-{{ .NewK3SVersion }}
	git checkout -B {{ .NewK8SVersion }}-{{ .NewK3SVersion }} upstream/{{.ReleaseBranch}}
	git clean -xfd
	
	sed -Ei "\|github.com/k3s-io/kubernetes| s|{{ .OldK8SVersion }}-{{ .OldK3SVersion }}|{{ .NewK8SVersion }}-{{ .NewK3SVersion }}|" go.mod
	sed -Ei "s/github.com\/k3s-io\/kubernetes/github.com\/%Handler%\/kubernetes/g" go.mod
	sed -Ei "s/k8s.io\/kubernetes v\S+/k8s.io\/kubernetes {{ .NewK8SVersion }}/" go.mod
	sed -Ei "s/{{ .OldK8SClient }}/{{ .NewK8SClient }}/g" go.mod # This should only change ~6 lines in go.mod
	
	go mod tidy
	# There is no need for running make since the changes will be only for go.mod
	# mkdir -p build/data && DRONE_TAG={{ .NewK8SVersion }}-{{ .NewK3SVersion }} make download && make generate

	git checkout -- */zz_generated*
	git add go.mod go.sum
	# This may add zz_generated* files that can corrupt the build. --all may be too aggressive.
	git commit --all --signoff -m "Update to {{ .NewK8SVersion }}"
	git push --set-upstream origin {{ .NewK8SVersion }}-{{ .NewK3SVersion }} # run git remote -v for your origin
	`

	updateScript = `
	#!/bin/bash
	cd %Workspace%/k3s
	git checkout {{ .NewK8SVersion }}-{{ .NewK3SVersion }}
	git clean -xfd
	
	sed -i "s/github.com\/%Handler%\/kubernetes/github.com\/k3s-io\/kubernetes/g" go.mod
	
	go mod tidy
	git commit --all --signoff -m "Update tags to k3s-io for {{ .NewK8SVersion }}"
	git push
	`
)
