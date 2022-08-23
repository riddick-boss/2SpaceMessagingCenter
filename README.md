# 2SpaceMessagingCenter

This repo is server side of [2Space](https://github.com/riddick-boss/2Space) android app.
It fetches info about upcoming launches from [TheSpaceDevs API](https://thespacedevs.com/llapi) and sends notifications to app if necessary, with usage of Firebase Cloud Messaging.

Used:
- Go
- Firebase Cloud Messaging

## Usage
 To use this code:
 - generate json with credentials in firebase console and place in it root named as "creds.json"
 - create file "topics.go" with following content
 ```go
 package main

func getReleaseTopicValue() string {
	return "YOUR_RELEASE_TOPIC"
}

func getDebugTopicValue() string {
	return "YOUR_DEBUG_TOPIC"
}
 
 ```
 
 - pass release/debug flag as command line param
 ```
 go run *.go DEBUG
 ```
 
 and you are ready to go :)
