#!/bin/sh
# note this version was written against gh cli version:
# gh version 2.23.0 (2023-02-07)
# https://github.com/cli/cli/releases/tag/v2.23.0

ghls() {
	_product="${1:-k3s}"
    _user="${2:-$GH_USERNAME}"
    case "$_product" in
    qa) gh issue ls -R rancher/qa-tasks -a "$_user" ;;
	rke2) gh issue ls -R rancher/rke2 -a "$_user" ;;
	k3s) gh issue ls -R k3s-io/k3s -a "$_user" ;;
	na)
	  printf "K3S-ISSUES---------------------------------------"
	  gh issue ls -R k3s-io/k3s 
	  printf "\nRKE2-ISSUES------------------------------------"
	  gh issue ls -R rancher/rke2
      printf "\nQA-ISSUES---------------------------------------"
      gh issue ls -R rancher/qa-tasks
	  ;;
	esac
	}

ghcom() {
	_product="${1:-k3s}"	
	_issue="${2}"
    case "$_product" in
	rke2)
	  gh issue view -c "$_issue" -R rancher/rke2
	  ;;	
	k3s)
	  gh issue view -c "$_issue" -R k3s-io/k3s
	  ;;
    qa)
      gh issue view -c "$_issue" -R rancher/qa-tasks
      ;;
	esac
	}

ghclosed() {
    _product="${1:-k3s}"
    _days="${2:-7}"
    _date=$(date -v -"$_days"d +%F)
    _milestone="${3:-v1.26.3+"$_product"}"
    case $_product in
    rke2)
      gh search issues --repo rancher/rke2 --closed \>"$_date" --milestone "$_milestone"
      ;;
    k3s)
        gh search issues --repo k3s-io/k3s --closed \>"$_date" --milestone "$_milestone"
        ;;
    qa)
        gh search issues --repo rancher/qa-tasks --closed \>"$_date"
        ;;
    esac
    }

    ghmile() {
     _milestone="${1:-v1.26.3}"
    _rke2_milestone="${_milestone}+rke2r1"
    _k3s_milestone="${_milestone}+k3s1"
    echo "$_rke2_milestone"
    echo "$_k3s_milestone"
    gh issue list -R rancher/rke2 --milestone "$_rke2_milestone";
    gh issue list -R k3s-io/k3s --milestone "$_k3s_milestone";
    }

# --- check CI for releases ---
get_ci() {
    #requires drone logins :/ 
    printf "https://drone-publish.rancher.io/rancher/rke2  or https://drone-publish.k3s.io/k3s-io/k3s"
}

# --- get four most recent branches ---

get_branches() {
    _product="${1:-$PRODUCT}"
    case "${_product}" in
    rke2)
        gh api https://api.github.com/repos/rancher/rke2/branches | jq '.[] | select(.name | test("release-1.*")) | .name' | tail -n 4
    ;;
    k3s)
        gh api https://api.github.com/repos/k3s-io/k3s/branches | jq '.[] | select(.name | test("release-1.*")) | .name' | tail -n 4
    ;;
    esac

}

# --- check DockerHub system agent installers and upgrade images ---
get_artifacts() {
    _product="${1:-$PRODUCT}"
    _branch="${2:-v1.26}"
    has_bin curl
    has_bin jq 
        printf "===== System Agent Installers ===== \n"
        curl -L -s "https://registry.hub.docker.com/v2/repositories/rancher/system-agent-installer-${_product}/tags?page_size=300" | jq -r ".results[].name" | sort --version-sort | grep -i -e "${_branch}" | tail -n 5 # > system-agent-installers.txt
        printf "===== Upgrade Images ===== \n"
        curl -L -s "https://registry.hub.docker.com/v2/repositories/rancher/${_product}-upgrade/tags?page_size=300" | jq -r ".results[].name" | sort --version-sort | grep -i -e "${_branch}" | tail -n 5 #  > upgrade-images.txt
        printf '===== %s Images ===== \n' "${_product}"
    case "${_product}" in
    rke2)
        curl -s -H "Accept: application/vnd.github+json" https://api.github.com/repos/rancher/rke2/releases | jq '.[].tag_name' | sort --version-sort | grep -i -e "${_branch}" | tail -n 1
        # diff --color -s --suppress-common-lines system-agent-installers.txt upgrade-images.txt
    ;;
    k3s)
        curl -s -H "Accept: application/vnd.github+json" https://api.github.com/repos/k3s-io/k3s/releases | jq '.[].tag_name' | sort --version-sort | grep -i -e "${_branch}" | tail -n 1
        # diff --color -s --suppress-common-lines system-agent-installers.txt upgrade-images.txt
    ;;
    esac
}
