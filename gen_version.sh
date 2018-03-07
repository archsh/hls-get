#!/usr/bin/env bash

echo "Generating Version $1 ..."
echo "/* AUTO GENERATED */" > $1
echo "package main" >> $1
echo "const (" >> $1
VERSION=`git describe --all | awk -F / '{ print $2 }'` && echo "    VERSION=\"$VERSION\"" >> $1
TAG=`git rev-parse HEAD` && echo "    TAG=\"$TAG\"" >> $1
DATE=`date +'%F %T'` && echo "    BUILD_TIME=\"$DATE\"" >> $1
echo ")" >> $1
