#!/bin/bash

mkdir -p ../src/usr/local/php/unraid-fileactivity/bin
go build
cp fileactivity-watcher ../src/usr/local/php/unraid-fileactivity/bin/