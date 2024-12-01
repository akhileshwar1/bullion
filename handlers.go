package main

import (
  "encoding/base64"
  "fmt"
  "log"
  "net/http"
  "os"
  "strings"
  "regexp"
  "strconv"
  "github.com/gin-gonic/gin"
  "google.golang.org/api/gmail/v1"
	"google.golang.org/api/sheets/v4"
)

func webhookHandler(gsrv *gmail.Service, ssrv *sheets.Service, historyBuffer *[]uint64, messageSet map[string]bool) gin.HandlerFunc {
  return func(c *gin.Context) {
    var pubSubMessage PubSubMessage
    if err := c.ShouldBindJSON(&pubSubMessage); err != nil {
      fmt.Printf("Error parsing Pub/Sub message: %v\n", err)
      c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
      return
    }

    decodedData, err := pubSubMessage.Message.DecodeData()
    if err != nil {
      fmt.Printf("Error decoding data: %v\n", err)
      c.JSON(http.StatusBadRequest, gin.H{"error": "invalid data"})
      return
    }

    fmt.Printf("Decoded data: %+v\n", decodedData)
    // Inputs
    user := "me"
    historyID := decodedData.HistoryID
    processHistory(gsrv, ssrv, user, historyID, historyBuffer, messageSet) // Use the passed srv
    c.JSON(http.StatusOK, gin.H{"status": "success"})
  }
}

func processHistory(gsrv *gmail.Service, ssrv *sheets.Service, user string, historyID uint64, historyBuffer *[]uint64, messageSet map[string]bool) error {
  oldestHistoryID := getOldestHistoryID(*historyBuffer)
  fmt.Println("in processing history!!")
  fmt.Println(oldestHistoryID)
  if oldestHistoryID == 0 {
    addToBuffer(historyBuffer, historyID)
    return fmt.Errorf("no historyId available to query")
  }

  // we don't start with the latest historyid since there will be no changes after that.
  historyCall := gsrv.Users.History.List(user).StartHistoryId(oldestHistoryID)
  response, err := historyCall.Do()
  if err != nil {
    return fmt.Errorf("error retrieving history: %v", err)
  }

  log.Println(messageSet)
  for _, history := range response.History {
    for _, msgAdded := range history.MessagesAdded {
      if !messageSet[msgAdded.Message.Id] {
        messageSet[msgAdded.Message.Id] = true
        err := processMessage(gsrv, ssrv, user, msgAdded.Message.Id)
        if err != nil {
          log.Printf("Error processing message: %v\n", err)
        }
      }
    }
  }

  // Add the latest historyId to the buffer.
  addToBuffer(historyBuffer, historyID)
  return nil
}

func processMessage(gsrv *gmail.Service, ssrv *sheets.Service, user, messageID string) error {
  messageCall := gsrv.Users.Messages.Get(user, messageID)
  messageResponse, err := messageCall.Do()
  if err != nil {
    return fmt.Errorf("unable to retrieve message: %v", err)
  }

  // Check if the email is from the expected sender
  if isEmailFrom(messageResponse, os.Getenv("EXPECTED_SENDER")) {
    // Extract the subject from the headers
    var subject string
    for _, header := range messageResponse.Payload.Headers {
      if header.Name == "Subject" {
        subject = header.Value
        break
      }
    }
    if subject == "" {
      return fmt.Errorf("subject not found in message headers")
    }

    // Extract the plain text body
    for _, part := range messageResponse.Payload.Parts {
      if part.MimeType == "text/plain" && part.Body.Data != "" {
        body, err := decodeBase64URL(part.Body.Data)
        if err != nil {
          return fmt.Errorf("unable to decode message body: %v", err)
        }

        // Pass the subject and body to the parseTransaction function
        transaction, err := parseTransaction(subject, body)
        if err != nil {
          return fmt.Errorf("unable to parse transaction: %v", err)
        }

        log.Printf("Parsed Transaction - Type: %s, Amount: %.2f\n", transaction.Type, transaction.Amount)
        if updateCashFlow(ssrv, os.Getenv("SPREADSHEET_ID"), os.Getenv("CF_SHEET_NAME"), *transaction) != nil {
          log.Println("Did not update sheets!!")
        }
      }
    }
  }

  return nil
}


func isEmailFrom(messageResponse *gmail.Message, expectedSender string) bool {
  var fromEmail string

  // Check the "From" field in the message headers
  for _, header := range messageResponse.Payload.Headers {
    if header.Name == "From" {
      fromEmail = header.Value
      break
    }
  }

  if fromEmail == "" {
    log.Println("Message does not have a 'From' field")
    return false
  }

  // Verify if the email is from the specified address
  if !strings.Contains(fromEmail, expectedSender) {
    log.Printf("Message is from %s, skipping\n", fromEmail)
    return false
  }

  return true
}

func parseTransaction(subject, body string) (*TransactionDetails, error) {
    // Determine transaction type from the subject
    subjectWords := strings.Fields(subject)
    if len(subjectWords) == 0 {
        return nil, fmt.Errorf("empty subject")
    }
    transactionType := subjectWords[0] // First word is either "Debit" or "Credit"

    if transactionType != "Debit" && transactionType != "Credit" {
        return nil, fmt.Errorf("unknown transaction type: %s", transactionType)
    }

    // Regex to extract the amount
    re := regexp.MustCompile(`INR\s([\d,]+(?:\.\d{2})?)`) // Matches patterns like "INR 999" or "INR 222.73"
    matches := re.FindStringSubmatch(body)
    if len(matches) < 2 {
        return nil, fmt.Errorf("amount not found in body")
    }

    // Convert the amount to float
    amountStr := strings.ReplaceAll(matches[1], ",", "") // Remove commas if present
    amount, err := strconv.ParseFloat(amountStr, 64)
    if err != nil {
        return nil, fmt.Errorf("invalid amount format: %v", err)
    }

    // Return parsed transaction details
    return &TransactionDetails{
        Type:   transactionType,
        Amount: amount,
    }, nil
}


func updateCashFlow(ssrv *sheets.Service, spreadsheetID string, sheetName string, transaction TransactionDetails) error {
    // Define the target cell range using sheetID and transaction type
    var cellRange string
    if transaction.Type == "Debit" {
        cellRange = fmt.Sprintf("%s!%s", sheetName, os.Getenv("CF_DEBIT_CELL"))
    } else if transaction.Type == "Credit" {
        cellRange = fmt.Sprintf("%s!%s", sheetName, os.Getenv("CF_CREDIT_CELL"))
    } else {
        return fmt.Errorf("unknown transaction type: %s", transaction.Type)
    }

    // Read the current value in the target cell
    readResp, err := ssrv.Spreadsheets.Values.Get(spreadsheetID, cellRange).Do()
    if err != nil {
        fmt.Println(err)
        return fmt.Errorf("unable to read cell value: %v", err)
    }

    // Parse the current value (default to 0 if the cell is empty)
    var currentValue float64
    if len(readResp.Values) > 0 && len(readResp.Values[0]) > 0 {
        currentValue, err = strconv.ParseFloat(readResp.Values[0][0].(string), 64)
        if err != nil {
            return fmt.Errorf("invalid number format in cell: %v", err)
        }
    }

    newValue := currentValue + transaction.Amount

    // Write the updated value back to the cell
    writeReq := &sheets.ValueRange{
        Values: [][]interface{}{{newValue}},
    }
    _, err = ssrv.Spreadsheets.Values.Update(spreadsheetID, cellRange, writeReq).ValueInputOption("RAW").Do()
    if err != nil {
        return fmt.Errorf("unable to update cell value: %v", err)
    }

    log.Printf("Updated %s cell (%s) with new value: %.2f\n", transaction.Type, cellRange, newValue)
    return nil
}


func decodeBase64URL(data string) (string, error) {
  decodedData, err := base64.URLEncoding.DecodeString(data)
  if err != nil {
    return "", err
  }
  return string(decodedData), nil
}
