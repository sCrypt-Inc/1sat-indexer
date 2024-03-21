#!/bin/bash

echo "Building bsv20-nofees ..."
cd cmd/bsv20-nofees && go build

echo "Building server ..."
cd ../server && go build

cd ../..