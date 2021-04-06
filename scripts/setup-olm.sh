#!/usr/bin/env bash

# # Try twice, since order matters
# kubectl apply -f https://github.com/operator-framework/operator-lifecycle-manager/releases/download/0.15.1/crds.yaml
# kubectl apply -f https://github.com/operator-framework/operator-lifecycle-manager/releases/download/0.15.1/olm.yaml

# # Delete "operatorhubio-catalog"
# kubectl delete catalogsource operatorhubio-catalog -n olm

if [ ! -f "./install-olm.sh" ]; then
    curl -L https://github.com/operator-framework/operator-lifecycle-manager/releases/download/v0.17.0/install.sh -o install-olm.sh
    chmod +x install-olm.sh
fi
./install-olm.sh v0.17.0
sleep 5

# Install OPM tool
if ! command -v opm &> /dev/null
then
    curl -o opm -L https://github.com/operator-framework/operator-registry/releases/download/v1.16.1/linux-amd64-opm
    chmod 755 opm
    sudo mv opm /usr/local/bin
fi