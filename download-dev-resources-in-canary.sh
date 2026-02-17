#!/bin/bash

# download resources from latest canary release
curl -L -H "Accept: application/vnd.github+json" -H "Authorization: Bearer $PAT" -H "X-GitHub-Api-Version: 2022-11-28" https://api.github.com/repos/mrdcvlsc/scheduling-system-backend/releases | grep browser_download_url | head -2 | grep -o 'https://[^"]*' | xargs -n 1 curl -L -O

# unzip dist.zip
unzip dist.zip -d ./

# unzip temporary data
unzip scheduling-system-temporary-data.zip -d ./

echo Done