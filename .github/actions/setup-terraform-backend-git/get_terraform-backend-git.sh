#!/bin/bash
if [[ -z "$INPUT_VERSION" ]]; then
  echo "Missing terraform-backend-git version information"
  exit 1
fi
terraform-backend-git version | grep "$INPUT_VERSION" &> /dev/null
if [ $? == 0 ]; then
   echo "terraform-backend-git $INPUT_VERSION is already installed! Exiting gracefully."
   exit 0
else
  echo "Installing terraform-backend-git to path."
fi

mkdir terraform-backend-git
TARGET_FILE="terraform-backend-git"
curl -LJ -o terraform-backend-git/$TARGET_FILE 'https://github.com/plumber-cd/terraform-backend-git/releases/download/'"$INPUT_VERSION"'/terraform-backend-git-darwin-386'
chmod +x terraform-backend-git/$TARGET_FILE
echo "terraform-backend-git" >> $GITHUB_PATH
echo "::set-output name=version::$INPUT_VERSION"