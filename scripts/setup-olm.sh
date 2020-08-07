#!/usr/bin/env bash

# # Try twice, since order matters
# kubectl apply -f https://github.com/operator-framework/operator-lifecycle-manager/releases/download/0.15.1/crds.yaml
# kubectl apply -f https://github.com/operator-framework/operator-lifecycle-manager/releases/download/0.15.1/olm.yaml

# # Delete "operatorhubio-catalog"
# kubectl delete catalogsource operatorhubio-catalog -n olm

if [ ! -f "./install-olm.sh" ]; then
    curl -L https://github.com/operator-framework/operator-lifecycle-manager/releases/download/0.15.1/install.sh -o install-olm.sh
    chmod +x install-olm.sh
fi
./install-olm.sh 0.15.1 