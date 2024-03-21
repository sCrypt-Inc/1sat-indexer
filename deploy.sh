#!/bin/bash
echo "Git pull latest ..."
git pull --rebase

sh ./build.sh

pm2 start ecosystem.config.js