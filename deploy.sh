#!/bin/bash
echo "Git pull latest ..."
git pull --rebase

echo "Building ..."
cd cmd/bsv20-nofees && go build

echo "Building ..."
cd ../server && go build

cd ../..

pm2 start ecosystem.config.js