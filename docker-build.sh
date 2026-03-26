#!/bin/bash
#set -x

DOCKER_NAME=iwfwebsolutions
BUILD_NAME=avast-rest
LATEST_TAG=latest
GIT_REPO=https://github.com/iwf-web/avast-rest
CODE_BASE=./code
DEF_PUSHDOCKERREGISTRY=0
DEF_BRANCH=main

function display_help() {
  echo -e "\nUsage: $0 -b|--branch branch [-pr | --pushregistry] [-h|--help]\n"
  echo "-b | --branch: branch to clone and build docker image with"
  echo "[-pr | --pushregistry]: automatically push image to docker registry (if build succeeded)"
  echo "[-h | --help]: this help information"
}

while :
do
    case "$1" in
      -b | --branch)
	  DEF_BRANCH="$2"
	  shift 2
	  ;;
      -pr | --pushregistry)
	  DEF_PUSHDOCKERREGISTRY=1
    shift
	  ;;
      --) # End of all options
	  shift
	  break
    ;;
      -h | --help)
	  display_help
	  # no shifting needed here, we're done.
	  exit 0
    ;;
      -*)
	  echo "Error: Illegal option: $1" >&2
    display_help
	  exit 1
	  ;;
      *)  # No more options
	  break
	  ;;
    esac
done

if [[ -n ${DEF_BRANCH} ]]; then
  if [ -d ${CODE_BASE} ]; then rm -rf ${CODE_BASE}; fi
  mkdir ${CODE_BASE}
  git clone --depth 1 -b ${DEF_BRANCH} ${GIT_REPO} ${CODE_BASE}

  # Start: get GIT information
  cd ./code

  REVISIONCOUNT=$(git log --oneline | wc -l | tr -d ' ')
  #PROJECTVERSION=$(git describe --tags --long)
  PROJECTVERSION=$(git describe --abbrev=0 --tags --exact-match)
  CLEANVERSION=${PROJECTVERSION%%-*}
  #echo "Full build: $projectversion-$revisioncount"
  GIT_BRANCH=$(git name-rev --name-only HEAD | sed "s/\^.*//")
  GIT_COMMIT=$(git rev-parse HEAD)
  GIT_COMMIT_SHORT=$(echo $GIT_COMMIT | head -c 8)
  GIT_DIRTY='false'
  BUILD_CREATOR=$(git config user.email)

  if [ ! -z $PROJECTVERSION ]; then
    BUILD_NUMBER=$PROJECTVERSION
    DOCKER_LATEST_TAG=${LATEST_TAG}
  else
    BUILD_NUMBER=${GIT_BRANCH}-$GIT_COMMIT_SHORT
    DOCKER_LATEST_TAG=${GIT_BRANCH}-latest
  fi


  # Whether the repo has uncommitted changes
  if [[ $(git status -s) ]]; then
      GIT_DIRTY='true'
  fi

  cd ..
  # End: get GIT information

  echo "===> Building and pushing docker image '$BUILD_NAME' with build '$BUILD_NUMBER' ($DOCKER_LATEST_TAG) ..."

  # Add "-q" for silence...
  docker buildx build \
    --platform=linux/amd64,linux/arm64 \
    --no-cache \
    --pull \
    --push \
    -f Dockerfile \
    -t $DOCKER_NAME/$BUILD_NAME:$DOCKER_LATEST_TAG \
    -t $DOCKER_NAME/$BUILD_NAME:$BUILD_NUMBER \
    --build-arg GIT_BRANCH="$GIT_BRANCH" \
    --build-arg GIT_COMMIT="$GIT_COMMIT" \
    --build-arg GIT_DIRTY="$GIT_DIRTY" \
    --build-arg BUILD_CREATOR="$BUILD_CREATOR" \
    --build-arg BUILD_NUMBER="$BUILD_NUMBER" \
    .

  echo "Done"

  if [ -d ${CODE_BASE} ]; then rm -rf ${CODE_BASE}; fi

else
  display_help
fi
