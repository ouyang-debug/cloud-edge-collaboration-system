package plus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// PublishMQTT publishes a message to the MQTT broker via the Base component's internal HTTP server
func PublishMQTT(topic string, payload interface{}) error {
	// payload can be string, []byte or struct (which will be JSON marshaled)
	var payloadStr string

	switch v := payload.(type) {
	case string:
		payloadStr = v
	case []byte:
		payloadStr = string(v)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %v", err)
		}
		payloadStr = string(b)
	}

	reqBody := map[string]string{
		"topic":   topic,
		"payload": payloadStr,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	// Use the internal HTTP server port 12346
	resp, err := http.Post("http://127.0.0.1:12346/mqtt/publish", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("publish failed with status: %s", resp.Status)
	}

	return nil
}
