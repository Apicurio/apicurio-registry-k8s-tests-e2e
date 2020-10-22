#!/bin/bash
set -e

TEST_ID=${TEST_ID:-apicurio}

LOGS_DEST=artifacts/$TEST_ID

echo "Copying tests logs to: $LOGS_DEST"

mkdir -p $LOGS_DEST
cp -r tests-logs $LOGS_DEST