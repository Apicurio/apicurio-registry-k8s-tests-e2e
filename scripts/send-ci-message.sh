#!/usr/bin/env bash

MESSAGE_FILE=$1

curl -i $CI_MESSAGES_ENDPOINT -X POST -H "Content-Type: application/json" --data-binary @$MESSAGE_FILE