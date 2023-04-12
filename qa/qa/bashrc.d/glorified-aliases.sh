#!/bin/sh
# ======================================================================================
# ================================== ~ ALIASES ~ =======================================
# ======================================================================================
komplete() {
    echo "source <(kubectl completion bash)" >> ~/.bashrc
}
#
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/go/bin:/snap/bin:/var/lib/rancher/k3s/bin:/var/lib/rancher/rke2/bin:/usr/local/bin/go/bin:/opt/rke2/bin/:${HOME}/.krew/bin
# enable color support of ls and also add handy aliases
if [ -x /usr/bin/dircolors ]; then
    test -r ~/.dircolors && eval "$(dircolors -b ~/.dircolors)" || eval "$(dircolors -b)"
    alias ls='ls --color=auto'
    alias dir='dir --color=auto'
    alias vdir='vdir --color=auto'
    alias grep='grep --color=auto'
    alias fgrep='fgrep --color=auto'
    alias egrep='egrep --color=auto'
fi
alias alert='notify-send --urgency=low -i "$([ $? = 0 ] && echo terminal || echo error)" "$(history|tail -n1|sed -e '\''s/^\s*[0-9]\+\s*//;s/[;&|]\s*alert$//'\'')"'
alias ll="ls -lahr "
alias eventz="kubectl get events -A "
alias k="kubectl "
alias kl="kubectl logs "
alias kg="kubectl get "
alias kgs="kubectl get svc "
alias kge="kubectl get endpoints "
alias kgn="kubectl get nodes "
alias kgp="kubectl get pods "
alias kga="kubectl get all -A -o wide "
alias w2="watch -n 3 "
alias sono="sonobuoy "
alias kd="kubectl describe "
alias srz="source ~/.bashrc"
alias ll='ls -alF'
alias la='ls -A'
alias l='ls -CF'

# --- wrap rke2 setup commands into one ---
setup_rke2() {
    get_rke2;
    set_figs rke2;
    set_etcduser;
    set_harden rke2;
    PRODUCT="rke2"
}

# --- wrap k3s setup commands into one ---
setup_k3s() {
    get_k3s;
    set_figs;
    set_etcduser;
    set_harden;
    PRODUCT="k3s"
}

# --- download k3s install.sh ---
get_k3s() {
    has_bin curl
    curl https://get.k3s.io --output install-k3s.sh
    sudo chmod +x install-k3s.sh
}

# --- install k3s server or agent and run---
go_k3s() {
    _version="${1}"
    #_CHANNEL="${3:-testing}"
    #_type="${4:-server}"
    sudo INSTALL_K3S_version="${_version}" INSTALL_K3S_EXEC=server ./install-K3s.sh
    #sudo INSTALL_"${product}"_version="$_version" INSTALL_"${product}"_CHANNEL="$CHANNEL" INSTALL_"${product}"_EXEC="${type}" ./install-"${product}".sh
}

# --- download rke2 install.sh ---
get_rke2() {
    has_bin curl
    curl https://get.rke2.io --output install-rke2.sh
    sudo chmod +x install-rke2.sh
}

# --- start rke2 systemctl service type ---
go_rke2() {
    _type="${1:-server}"
    has_bin rke2
    sudo systemctl enable rke2-"${_type}" --now
}

# --- prints out node token ---
get_token() {
    _product="${1:-$PRODUCT}"
    sudo cat /var/lib/rancher/"${_product}"/server/node-token
}

# --- print to console current config file ---
get_figs() {
    _product="${1:-$PRODUCT}"
    printf '=========== %s config =========== \n' "${_product}"
    sudo cat /etc/rancher/"${_product}"/config.yaml;
}

# --- restore previously saved config file (from void function) ---
go_replay() {
    # after you've called void the tmp-confs directory is made with your previous config
    _product="${1:-$PRODUCT}"
    sudo mkdir -p /etc/rancher/"${_product}"/;
    sudo cp ~/tmp-confs/"${_product}"-config.yaml /etc/rancher/"${_product}"/config.yaml
}

# --- set KUBECONFIG environment variable ---
set_kubefig() {
    _product="${1:-$PRODUCT}"
    #consider the flag rootless instead of passing t or f
    _rootless="${2:-false}"
    if [ "${_rootless}" = true ]; then
        export KUBECONFIG=/home/"${USER}"/.kube/k3s.yaml
    else
        export KUBECONFIG=/etc/rancher/"${_product}"/"${_product}".yaml
        sudo chmod 644 /etc/rancher/"${_product}"/"${_product}".yaml
    fi
}

# --- check etcdctl secrets ---
get_secret() {
    has_bin etcdctl
     _product="${1:-$PRODUCT}"
    _secret="${2:-secret1}"
    sudo ETCDCTL_API=3 etcdctl \
    --cert /var/lib/rancher/"${_product}"/server/tls/etcd/server-client.crt \
    --key /var/lib/rancher/"${_product}"/server/tls/etcd/server-client.key \
    --endpoints https://127.0.0.1:2379 \
    --cacert /var/lib/rancher/"${_product}"/server/tls/etcd/server-ca.crt \
    get /registry/secrets/default/"${_secret}" | hexdump -C
}

# --- list binaries in directory from product install ---
get_bins() {
    _product="${1:-$PRODUCT}"
    ls /var/lib/rancher/"${_product}"/data/current/bin/
    #consider _variable to find the embedded bin at known directory locations
    echo "/var/lib/rancher/k3s/data/current/bin OR /var/lib/rancher/rke2/data/current/bin"
}

# --- get performance profile from kubectl api ---
get_pprof() {
    has_bin curl
    # note this requires enable-pprof=true in config.yaml
    set -- "profile" "symbol" "trace?seconds=8" "cmdline"
    for id; do
        curl --insecure https://localhost:6443/debug/pprof/"${id}" > ~/"${id}".pprof
    done
        #back on localhost or wherever GoLang is installed 
        # run $ go tool pprof GENERATED_FILENAME to run various commands on the files like .PNG etc
        # trace uses $ go tool trace trace-outputfile view refresher - https://github.com/k3s-io/k3s/pull/5527 /// https://pkg.go.dev/net/http/pprof
}

# --- start sonobuoy e2e conformance tests ---
go_sono() {
    _product="${1:-$PRODUCT}"
    _version
    _version=$("${_product}" --version | awk 'NR<2{print $3}' | cut -c -7)
    has_bin sonobuoy
    #sonobuoy run --wait --kubeconfig /etc/rancher/"${product}"/"${product}".yaml --plugin https://raw.githubusercontent.com/vmware-tanzu/sonobuoy-plugins/master/cis-benchmarks/kube-bench-plugin.yaml --plugin https://raw.githubusercontent.com/vmware-tanzu/sonobuoy-plugins/master/cis-benchmarks/kube-bench-master-plugin.yaml
    sonobuoy run --wait --kubeconfig /etc/rancher/"${_product}"/"${_product}".yaml --kubernetes-version="${_version}" --mode=certified-conformance
}

# --- view sonobuoy results ---
sono_results() {
    _product="${1:-$PRODUCT}"
    has_bin sonobuoy
    _results=$(sonobuoy retrieve --kubeconfig /etc/rancher/"${_product}"/"${_product}".yaml)
    sonobuoy results "${_results}"    
}

# --- removes all docker images from the _cache ---
void_docker() {
    has_bin docker
    docker rmi "$(docker images -a -q)"
}

# --- requires helm --- REWRITE WITH INSTALL.SH IF CHECKS FOR HELM --- https://github.com/k3s-io/k3s/blob/master/install.sh
make_ranch() {
    has_bin helm
    has_bin kubectl
    helm repo add rancher-latest https://releases.rancher.com/server-charts/latest
    helm repo add jetstack https://charts.jetstack.io
    helm repo update
    kubectl apply --validate=false -f https://github.com/jetstack/cert-manager/releases/download/v1.8.0/cert-manager.crds.yaml
    kubectl create namespace cattle-system
    wait
    helm install cert-manager jetstack/cert-manager -n cert-manager --create-namespace --version v1.8.0
    wait
    echo "checking cert-manager pods.... "
    kubectl get pods -n cert-manager
    wait
    helm install rancher rancher-latest/rancher -n cattle-system --set hostname=break.qa.web --set rancherImageTag=v2.7-head --version=v2.7-head 
    #wait
    watch -n 7 kubectl -n cattle-system rollout status deploy/rancher
    #kubectl port-forward "$(kubectl get pods --selector "app.kubernetes.io/name=traefik" --output=name -A)" 9000:9000 -A
}

# --- gets rancher password and dashboard url ---
get_ranch() {
    has_bin kubectl
    echo https://break.qa/dashboard/?setup="$(kubectl get secret --namespace cattle-system bootstrap-secret -o go-template='{{.data.bootstrapPassword|base64decode}}')"
    kubectl get secret --namespace cattle-system bootstrap-secret -o go-template='{{.data.bootstrapPassword|base64decode}}{{ "\n" }}'
}

# --- forward the rancher ui to all interfaces ---
forward_ranch() {
    has_bin kubectl
    kubectl port-forward -n cattle-system svc/rancher 9944:443 --address='0.0.0.0'
}

# --- label nodes server and agent respectively for suc upgrade to run correctly ---
set_labels() {
    _product="${1:-$PRODUCT}"
    has_bin kubectl
    kubectl label node -l node-role.kubernetes.io/master==true "${_product}"-upgrade=server 
    kubectl label node -l node-role.kubernetes.io/master!=true "${_product}"-upgrade=agent
}

# --- print info logs when debug is enabled ---
get_logs() {
    _product="${1:-$PRODUCT}"
    _type="${2:-server}"
    sudo journalctl -xeu "${_product}"-"${_type}" -o json-pretty | grep -i -e message -e info
    # experiment with de-cluttering the normal pings to tunnel server etc
}

# --- rotate and vacuum logs ---
clean_logs() {
    sudo journalctl --rotate && sudo journalctl --vacuum-time=1s
}

# --- helper functions for logs ---
get_ulog() {
    sudo cat /var/log/ulog/syslogemu.log
}

# --- get systemctl status ---
get_status() {
    _product="${1:-$PRODUCT}"
    sudo systemctl status "${_product}"
}

# --- quickly list etcd cluster members ---
get_etcd() {
    _product="${1:-rke2}"
    has_bin etcdctl
     sudo ETCDCTL_API=3 etcdctl \
    --cert /var/lib/rancher/"${_product}"/server/tls/etcd/server-client.crt \
    --key /var/lib/rancher/"${_product}"/server/tls/etcd/server-client.key \
    --endpoints https://127.0.0.1:2379 \
    --cacert /var/lib/rancher/"${_product}"/server/tls/etcd/server-ca.crt \
    member list -w table
    ##   https://etcd.io/docs/v3.5/tutorials/how-to-deal-with-membership/
}

# --- quickly list manifest files ---
get_manifests() {
    _product="${1:-$PRODUCT}"
    _command="ls --color=auto /var/lib/rancher/${_product}/server/manifests"
    has_bin bash
    sudo --preserve-env=product bash -c "${_command}"
}

# --- quickly exec a shell in a pod ---
k_tty() {
    _namespace="{$1}"
    _podTarget="{$2}"
    has_bin kubectl
    kubectl exec -n "${_namespace}" --stdin --tty "${_podTarget}" -- /bin/bash
}

# --- quickly list any pods in any namespace in error or crashloop status ---
get_podcrash() {
    has_bin kubectl
    kubectl get pods -A | awk '$3 ~/CrashLoopBackOff/ {print $1 $3}'
    kubectl get pods -A | awk '$3 ~/Error/ {print $1 $3}'
}


# --- quickly list node taints ---
get_taints() {
    has_bin kubectl
    kubectl get nodes -o custom-columns=NAME:.metadata.name,TAINTS:.spec.taints
}
#

# --- check rke2 crictl images and versions ---
get_images() {
    _product="${1:-$PRODUCT}"
    case "${_product}" in
    rke2) sudo /var/lib/rancher/"${_product}"/bin/crictl --config /var/lib/rancher/"${_product}"/agent/etc/crictl.yaml ps
          sudo /var/lib/rancher/"${_product}"/bin/crictl --config /var/lib/rancher/"${_product}"/agent/etc/crictl.yaml images
        ;;
    k3s) 
        sudo k3s crictl ps
        sudo k3s crictl img ls
        ;;
    esac
}

# --- gets the ips of the pods in the provided namespace ---
get_podips() {
    _namespace="${1:-kube-system}"
    kubectl get pods -n "${_namespace}" -o jsonpath='{range .items[*]}{.metadata.name}{"    "}{.status.podIP}{"\n"}{end}'
}

# --- this is a work in progress but it gets the ips on the pods and pings the adjacent pods to check network connectivity this will likely go away in this form ---
ping_pods() {
    for pod in $(kubectl get pods -A | awk 'NR>1{print $2}')
        do   
            for ip in $(kubectl get pods -n kube-system "${pod}" -o yaml | grep -e "podIP: " | awk '{print $2}')
                do printf '%s' "${pod} \n"
                kubectl exec -n kube-system -it "${pod}" -- ping -c 5 "${ip}"
                done
        done
}

# --- less typing get ids of containers ---
get_containerids() {
    sudo ctr --address /run/k3s/containerd/containerd.sock --namespace k8s.io c ls
}

# --- less typing get containerd version ---
get_contd() {
    sudo /var/lib/rancher/rke2/bin/containerd --version
}

# --- take an adhoc etcd snapshot with less typing ---
take_etcd() {
    _folder="${1:-123}"
    sudo rke2 etcd-snapshot --s3 --s3-bucket=YOUR_BUCKET --s3-folder="${_folder}" --s3-region=YOUR_REGION --s3-access-key=YOUR_ACCESS_KEY --s3-secret-key=YOUR_SECRET_KEY
}

# --- killall uninstall all ---
void() {
    if [ ! -d ~/tmp-confs/ ]; then
        mkdir ~/tmp-confs/
    fi

    if command -v k3s > /dev/null 2>&1; then
        binary_path="$(command -v k3s)"
        binary_name="k3s"
    elif command -v rke2 > /dev/null 2>&1; then
        binary_path="$(command -v rke2)"
        binary_name="rke2"
    else
        echo "Neither k3s nor rke2 are installed"
    fi

    sudo cp "/etc/rancher/${binary_name}/config.yaml" ~/tmp-confs/"${binary_name}"-config.yaml
    binary_path="${binary_path%/*}"
    if pgrep -f "${binary_name}-agent" > /dev/null; then
        sudo "${binary_path}"/"${binary_name}"-killall.sh
        sudo "${binary_path}"/"${binary_name}"-agent-uninstall.sh
    else
        sudo "${binary_path}"/"${binary_name}"-killall.sh
        sudo "${binary_path}"/"${binary_name}"-uninstall.sh
    fi
    echo "${binary_path}"
}
