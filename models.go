package main

import (
  "log"
	"encoding/base64"
	"encoding/json"
)

type PubSubMessage struct {
	Message     MessageData `json:"message"`
	Subscription string      `json:"subscription"`
}

type MessageData struct {
	Data        string            `json:"data"`
	MessageID   string            `json:"messageId"`
	PublishTime string            `json:"publishTime"`
}

type DecodedData struct {
	EmailAddress string `json:"emailAddress"`
	HistoryID    int `json:"historyId"`
}

func (m *MessageData) DecodeData() (DecodedData, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(m.Data)
	if err != nil {
		return DecodedData{}, err
	}
	var data DecodedData
	if err := json.Unmarshal(decodedBytes, &data); err != nil {
    log.Printf("Error unmarshalling JSON: %v\n", err)
		return DecodedData{}, err
	}
	return data, nil
}
