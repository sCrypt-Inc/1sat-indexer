#!/bin/bash
echo "Git pull latest ..."
git pull --rebase

if [ -e .env ] 
then
  source .env
  echo "load .env ..."
fi

echo "Kill bsv20-nofees ..."
pkill bsv20-nofees

echo "Building ..."
cd cmd/bsv20-nofees && go build

echo "Start bsv20-nofees ..."
./bsv20-nofees -t "$FULL_SUBSCRIPTIONID" -s 1600000 > output.log 2>&1 &


echo "Kill server ..."
pkill server

echo "Building ..."
cd ../server && go build

echo "Start server ..."
./server -p 8082 > output.log 2>&1 &