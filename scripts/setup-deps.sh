if ! command -v kind &> /dev/null
then
    echo "Installing kind"
    curl -Lo ./kind "https://kind.sigs.k8s.io/dl/v0.8.1/kind-$(uname)-amd64"
    chmod +x ./kind
fi

echo "Installing Antora"
sudo npm i -g @antora/cli @antora/site-generator-default
antora -v