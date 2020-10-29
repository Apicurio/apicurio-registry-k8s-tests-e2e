#!/bin/bash
set +x

if [ -z "$CLUSTER_CA_CERT_SECRET" ]; then
    echo "missing env var CLUSTER_CA_CERT_SECRET"
    exit 1
fi
echo $CLUSTER_CA_CERT_SECRET

echo $CLIENT_CERT_SECRET

if [ -z "$TRUSTSTORE_SECRET" ]; then
    echo "missing env var TRUSTSTORE_SECRET"
    exit 1
fi
echo $TRUSTSTORE_SECRET

echo $KEYSTORE_SECRET

echo $HOSTNAME

# Parameters:
# $1: Path to the new truststore
# $2: Truststore password
# $3: Public key to be imported
# $4: Alias of the certificate
function create_truststore {
   keytool -keystore $1 -storepass $2 -noprompt -alias $4 -import -file $3 -storetype PKCS12
}

# Parameters:
# $1: Path to the new keystore
# $2: Truststore password
# $3: Public key to be imported
# $4: Private key to be imported
# $5: Alias of the certificate
function create_keystore {
   RANDFILE=/tmp/.rnd openssl pkcs12 -export -in $3 -inkey $4 -name $HOSTNAME -password pass:$2 -out $1
}

CLUSTER_CA_CRT=$(oc get secret $CLUSTER_CA_CERT_SECRET -o 'go-template={{index .data "ca.crt"}}' | base64 -d -)

echo "Preparing truststore"
export TRUSTSTORE_PASSWORD=$(< /dev/urandom tr -dc _A-Z-a-z-0-9 | head -c32)
echo "$CLUSTER_CA_CRT" > /tmp/ca.crt
create_truststore /tmp/truststore.p12 $TRUSTSTORE_PASSWORD /tmp/ca.crt ca

# ca.p12 ca.password
oc create secret generic $TRUSTSTORE_SECRET --from-file=ca.p12=/tmp/truststore.p12 --from-literal=ca.password=$TRUSTSTORE_PASSWORD

###

if [[ "$CLIENT_CERT_SECRET" ]];
then
    CLIENT_CRT=$(oc get secret $CLIENT_CERT_SECRET -o 'go-template={{index .data "user.crt"}}' | base64 -d -)
    CLIENT_KEY=$(oc get secret $CLIENT_CERT_SECRET -o 'go-template={{index .data "user.key"}}' | base64 -d -)

    echo "Preparing keystore"
    export KEYSTORE_PASSWORD=$(< /dev/urandom tr -dc _A-Z-a-z-0-9 | head -c32)
    echo "$CLIENT_CRT" > /tmp/user.crt
    echo "$CLIENT_KEY" > /tmp/user.key

    create_keystore /tmp/keystore.p12 $KEYSTORE_PASSWORD /tmp/user.crt /tmp/user.key

    # user.p12 user.password
    oc create secret generic $KEYSTORE_SECRET --from-file=user.p12=/tmp/keystore.p12 --from-literal=user.password=$KEYSTORE_PASSWORD
fi