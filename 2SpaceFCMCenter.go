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
	"time"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"google.golang.org/api/option"
)

const UPCOMING_LAUNCH_URL = "https://ll.thespacedevs.com/2.2.0/launch/upcoming/?limit=1&is_crewed=false&include_suborbital=true&related=false&hide_recent_previous=true"
const TEN_MINUTES_IN_SECONDS = 600
const NOTIFICATION_ID_KEY = "all_launches_notification_id"
const NOTIFICATION_TITLE = "Upcoming launch"

func setupFcmClient() (context.Context, *messaging.Client) {
	ctx := context.Background()
	opts := []option.ClientOption{option.WithCredentialsFile("creds.json")}

	app, _ := firebase.NewApp(ctx, nil, opts...)

	fcmClient, _ := app.Messaging(ctx)

	return ctx, fcmClient
}

// returns: shouldSendNotification, launch id, launch name
func getInfoAboutUpcomingLaunch() (bool, string, string) {
	response, _ := http.Get(UPCOMING_LAUNCH_URL)
	body, _ := ioutil.ReadAll(response.Body)

	var result map[string]interface{}

	json.Unmarshal([]byte(body), &result)

	launch := result["results"].([]interface{})[0].(map[string]interface{})

	status := launch["status"].(map[string]interface{})
	abbrev := status["abbrev"].(string)
	isLaunchReady := abbrev == "Go"

	if !isLaunchReady {
		return false, "", ""
	}

	id := launch["id"].(string)
	name := launch["name"].(string)

	nowInSeconds := time.Now().Unix()

	windowStartTimeInSeconds := convertTimeStampToSeconds(launch["window_start"].(string))
	isWindowStartInRange := isTimeInRange(windowStartTimeInSeconds, nowInSeconds)

	windowEndTimeInSeconds := convertTimeStampToSeconds(launch["window_end"].(string))
	isWindowEndInRange := isTimeInRange(windowEndTimeInSeconds, nowInSeconds)

	shouldSendNotification := isLaunchReady && (isWindowStartInRange || isWindowEndInRange)

	return shouldSendNotification, id, name
}

func convertTimeStampToSeconds(timestamp string) int64 {
	time, _ := time.Parse(time.RFC3339, timestamp)
	timeInSeconds := time.Unix()
	return timeInSeconds
}

func isTimeInRange(timeInSeconds int64, nowInSeconds int64) bool {
	diff := timeInSeconds - nowInSeconds
	return timeInSeconds > 0 && diff < TEN_MINUTES_IN_SECONDS
}

func createNotificationBody(launchName string) string {
	return launchName + " is expected to launch in next 10 minutes"
}

func createNotification(id string, launchName string, topic string) *messaging.Message {
	body := createNotificationBody(launchName)

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

func sendNotification(ctx context.Context, client *messaging.Client, notification *messaging.Message) {

	response, _ := client.Send(ctx, notification)
	fmt.Println("Successfully sent notification:", response)
}

func runInfinite(ctx context.Context, client *messaging.Client) {

	for {
		shouldSendNotification, launchId, launchName := getInfoAboutUpcomingLaunch()
		if shouldSendNotification && launchId != "" && launchName != "" {
			topic := getTopicValue()
			notification := createNotification(launchId, launchName, topic)
			sendNotification(ctx, client, notification)
		}

		time.Sleep(10 * time.Minute)
	}
}

func main() {
	fmt.Println("Launching 2SpaceFcmMessagingCenter...")
	ctx, fcmClient := setupFcmClient()
	runInfinite(ctx, fcmClient)
}
