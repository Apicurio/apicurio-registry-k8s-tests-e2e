if ! command -v kind &> /dev/null
then
    echo "Installing kind"
    curl -Lo ./kind "https://kind.sigs.k8s.io/dl/v0.29.0/kind-$(uname)-amd64"
    chmod +x ./kind
fi

echo "Installing Antora"
sudo npm i -g @antora/cli@3.0.0 @antora/site-generator@3.0.0
antora -v