/*
 * Created Date: Sunday, July 24th 2022
 * Author: Pawel Kremienowski
 *
 */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"google.golang.org/api/option"
)

const UPCOMING_LAUNCH_URL = "https://ll.thespacedevs.com/2.2.0/launch/upcoming/?limit=1&is_crewed=false&include_suborbital=true&related=false&hide_recent_previous=true"
const TEN_MINUTES_IN_SECONDS = 600
const NOTIFICATION_ID_KEY = "all_launches_notification_id"
const NOTIFICATION_TITLE = "Upcoming launch"

const RELEASE_FLAG = "RELEASE"
const DEBUG_FLAG = "DEBUG"

// global varibles
var lastNotificationLaunchId = ""

func setupFcmClient() (context.Context, *messaging.Client, error) {
	ctx := context.Background()
	opts := []option.ClientOption{option.WithCredentialsFile("creds.json")}

	app, err1 := firebase.NewApp(ctx, nil, opts...)
	if err1 != nil {
		return nil, nil, err1
	}

	fcmClient, err2 := app.Messaging(ctx)
	if err2 != nil {
		return nil, nil, err2
	}

	return ctx, fcmClient, nil
}

// returns: shouldSendNotification, launch id, launch name, window start
func getInfoAboutUpcomingLaunch() (bool, string, string, string) {
	response, err1 := http.Get(UPCOMING_LAUNCH_URL)
	if err1 != nil || response == nil {
		return false, "", "", ""
	}

	body, err2 := ioutil.ReadAll(response.Body)
	if err2 != nil || body == nil {
		return false, "", "", ""
	}

	var result map[string]interface{}

	err3 := json.Unmarshal([]byte(body), &result)
	if err3 != nil {
		return false, "", "", ""
	}

	launch := result["results"].([]interface{})[0].(map[string]interface{})

	status := launch["status"].(map[string]interface{})
	abbrev := status["abbrev"].(string)
	isLaunchReady := abbrev == "Go"

	if !isLaunchReady {
		return false, "", "", ""
	}

	id := launch["id"].(string)
	name := launch["name"].(string)

	nowInSeconds := time.Now().Unix()

	windowStart := launch["window_start"].(string)
	windowStartTimeInSeconds := convertTimeStampToSeconds(windowStart)
	isWindowStartInRange := isTimeInRange(windowStartTimeInSeconds, nowInSeconds)

	windowEndTimeInSeconds := convertTimeStampToSeconds(launch["window_end"].(string))
	isWindowEndInRange := isTimeInRange(windowEndTimeInSeconds, nowInSeconds)

	shouldSendNotification := isLaunchReady && (isWindowStartInRange || isWindowEndInRange)

	return shouldSendNotification, id, name, windowStart
}

func convertTimeStampToSeconds(timestamp string) int64 {
	time, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return 0
	}
	timeInSeconds := time.Unix()
	return timeInSeconds
}

func isTimeInRange(timeInSeconds int64, nowInSeconds int64) bool {
	diff := timeInSeconds - nowInSeconds
	return timeInSeconds > 0 && diff < TEN_MINUTES_IN_SECONDS && diff > 0
}

func createNotificationBody(launchName string, windowStart string) string {
	return launchName + " went to space (window start: " + windowStart + ")"
}

func createNotification(id string, launchName string, windowStart string, topic string) *messaging.Message {
	body := createNotificationBody(launchName, windowStart)

	message := &messaging.Message{
		Data: map[string]string{
			NOTIFICATION_ID_KEY: id,
		},
		Notification: &messaging.Notification{
			Title: NOTIFICATION_TITLE,
			Body:  body,
		},
		Topic: topic,
	}

	return message
}

func sendNotification(ctx context.Context, client *messaging.Client, notification *messaging.Message, launchId string) {

	response, err := client.Send(ctx, notification)
	if err != nil {
		fmt.Println("Failed to send message")
		return
	}
	fmt.Println("Updating last notification id, was:", lastNotificationLaunchId)
	lastNotificationLaunchId = launchId
	fmt.Println("Last notification id updated, is:", lastNotificationLaunchId)
	fmt.Println("Successfully sent notification:", response)
}

func runInfinite(ctx context.Context, client *messaging.Client, topic string) {

	for {
		shouldSendNotification, launchId, launchName, windowStart := getInfoAboutUpcomingLaunch()
		if shouldSendNotification && launchId != "" && launchId != lastNotificationLaunchId && launchName != "" {
			notification := createNotification(launchId, launchName, windowStart, topic)
			sendNotification(ctx, client, notification, launchId)
		}

		time.Sleep(10 * time.Minute)
	}
}

func prepareTopic() string {
	topic := ""

	switch {
	case len(os.Args) == 1: // if no params were passed
		panic("Specify version either: DEBUG or RELEASE")
	case os.Args[1] == DEBUG_FLAG:
		topic = getDebugTopicValue()
	case os.Args[1] == RELEASE_FLAG:
		topic = getReleaseTopicValue()
	default:
		panic("Specify version either: DEBUG or RELEASE")
	}

	return topic
}

func main() {
	fmt.Println("Launching 2SpaceFcmMessagingCenter...")
	ctx, fcmClient, err := setupFcmClient()
	if err != nil {
		fmt.Println("Failed to initialize FcmClient")
		return
	}

	topic := prepareTopic()

	runInfinite(ctx, fcmClient, topic)
}
