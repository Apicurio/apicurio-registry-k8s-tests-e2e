if ! command -v kind &> /dev/null
then
    echo "installing kind"
    curl -Lo ./kind "https://kind.sigs.k8s.io/dl/v0.15.0/kind-$(uname)-amd64"
    chmod +x ./kind
fi