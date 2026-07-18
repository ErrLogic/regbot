#!/bin/bash
export ANDROID_HOME=/usr/lib/android-sdk
export ANDROID_SDK_ROOT=/usr/lib/android-sdk
appium --log-level error --log /tmp/appium.log &
echo "Appium starting with ANDROID_HOME=$ANDROID_HOME"
sleep 3
curl -s http://127.0.0.1:4723/status
