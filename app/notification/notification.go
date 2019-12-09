/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
*/

package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/edgexfoundry/app-functions-sdk-go/appcontext"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/notifications"
	"github.impcloud.net/RSP-Inventory-Suite/loss-prevention-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/helper"
	"net/http"
	"strconv"

	log "github.com/sirupsen/logrus"
)

const (
	notificationSlug     = "loss-prevention-service"
	subscriptionEndpoint = "/api/v1/subscription"
	notificationCategory = "SECURITY"
	notificationSeverity = "CRITICAL"
	notificationLabel    = "LOSS-PREVENTION"
	notificationSender   = "Loss Prevention App"
)

// Subscriber holds the body schema to register a subscriber to EdgeX
type Subscriber struct {
	Slug                 string     `json:"slug"`
	Receiver             string     `json:"receiver"`
	SubscribedCategories []string   `json:"subscribedCategories"`
	SubscribedLabels     []string   `json:"subscribedLabels"`
	Channels             []Channels `json:"channels"`
}

// Channels holds the body schema to specify different type of notification channels (email, SMS, REST post call)
type Channels struct {
	Type          string   `json:"type"`
	URL           string   `json:"url,omitempty"`
	MailAddresses []string `json:"mailAddresses,omitempty"`
}

// This leverages EdgeX Alerts & notification service
func PostNotification(edgexcontext *appcontext.Context, content string) error {

	log.Info("Sending notification to EdgeX...")

	notif := notifications.Notification{
		Slug:     notificationSlug + "-" + strconv.FormatInt(helper.UnixMilliNow(), 10),
		Labels:   []string{notificationLabel},
		Sender:   notificationSender,
		Category: notificationCategory,
		Severity: notificationSeverity,
		Content:  content,
	}

	err := edgexcontext.NotificationsClient.SendNotification(notif, context.Background())
	if err != nil {
		log.Errorf("unable to post notification to EdgeX: %v", err)
	} else {
		log.Info("Successfully sent notification to EdgeX")
	}
	return err
}

// RegisterSubscriber registers a subscriber to EdgeX Alerts & notification service using email as channel
func RegisterSubscriber(emails []string) error {

	// Create requestBody
	subscriber := new(Subscriber)
	channels := Channels{Type: "EMAIL", MailAddresses: emails}

	subscriber.Slug = notificationSlug
	subscriber.Receiver = "USER"
	subscriber.SubscribedCategories = []string{notificationCategory}
	subscriber.SubscribedLabels = []string{notificationLabel}
	subscriber.Channels = []Channels{channels}

	requestBody, err := json.Marshal(subscriber)
	if err != nil {
		return err
	}

	response, err := http.Post(config.AppConfig.NotificationServiceURL+subscriptionEndpoint, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated && response.StatusCode != http.StatusConflict {
		return fmt.Errorf("POST error on subscriber endpoint, StatusCode %d", response.StatusCode)
	}

	return nil

}
