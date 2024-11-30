package main

import (
  "log"
	"encoding/base64"
	"encoding/json"
)

type TransactionDetails struct {
    Type   string  // "Debit" or "Credit"
    Amount float64 // Transaction amount
}

type HistoryResponse struct {
	History []History `json:"history"`
}

type History struct {
	MessagesAdded []MessagesAdded `json:"messagesAdded"`
}

type MessagesAdded struct {
	Message Message `json:"message"`
}

// Message represents a Gmail message.
type Message struct {
	ID string `json:"id"`
}

// MessageResponse represents the response from the Gmail Message API.
type MessageResponse struct {
	Payload Payload `json:"payload"`
}

type Payload struct {
	Parts []Part `json:"parts"`
}

type Part struct {
	MimeType string `json:"mimeType"`
	Body     Body   `json:"body"`
}

type Body struct {
	Data string `json:"data"`
}

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
	HistoryID    uint64 `json:"historyId"`
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
