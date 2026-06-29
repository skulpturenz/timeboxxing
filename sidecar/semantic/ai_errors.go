package semantic

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

const maxAIErrorDetailLength = 240

type AIRequestError struct {
	Provider     Provider
	Endpoint     string
	StatusCode   int
	Message      string
	ErrorType    string
	RetryAfter   string
	ResponseBody string
	TransportErr error
}

func (e *AIRequestError) Error() string {
	if e == nil {
		return ""
	}
	if e.TransportErr != nil {
		return fmt.Sprintf("%s request to %s failed: %v", providerLabel(e.Provider), e.Endpoint, e.TransportErr)
	}
	if e.StatusCode > 0 {
		message := strings.TrimSpace(e.Message)
		if message == "" {
			message = strings.TrimSpace(e.ResponseBody)
		}
		if message == "" {
			message = http.StatusText(e.StatusCode)
		}
		return fmt.Sprintf("%s request to %s failed with status %d: %s", providerLabel(e.Provider), e.Endpoint, e.StatusCode, truncateAIErrorDetail(message))
	}
	return fmt.Sprintf("%s request to %s failed", providerLabel(e.Provider), e.Endpoint)
}

func AIRequestUserMessage(err error) (string, bool) {
	requestErr, ok := findAIRequestError(err)
	if !ok {
		return "", false
	}

	return requestErr.UserMessage(), true
}

func AIRequestStatusCode(err error) (int, bool) {
	requestErr, ok := findAIRequestError(err)
	if !ok || requestErr.StatusCode == 0 {
		return 0, false
	}
	return requestErr.StatusCode, true
}

func (e *AIRequestError) UserMessage() string {
	if e == nil {
		return "AI provider request failed."
	}
	label := providerLabel(e.Provider)
	if e.TransportErr != nil {
		return fmt.Sprintf("%s could not be reached. Check the base URL and network connection.", label)
	}

	detail := truncateAIErrorDetail(strings.TrimSpace(e.Message))
	switch e.StatusCode {
	case http.StatusUnauthorized:
		return fmt.Sprintf("%s API key is invalid or expired.", label)
	case http.StatusPaymentRequired:
		return fmt.Sprintf("%s account has insufficient credits.", label)
	case http.StatusBadRequest, http.StatusNotFound:
		if detail != "" {
			return fmt.Sprintf("Selected %s model is unavailable or the request is invalid: %s", label, detail)
		}
		return fmt.Sprintf("Selected %s model is unavailable or the request is invalid.", label)
	case http.StatusTooManyRequests:
		if retry := retryAfterPhrase(e.RetryAfter); retry != "" {
			return fmt.Sprintf("%s rate limit exceeded. Try again %s.", label, retry)
		}
		return fmt.Sprintf("%s rate limit exceeded. Try again in a moment.", label)
	case http.StatusBadGateway, http.StatusServiceUnavailable, 529:
		return fmt.Sprintf("%s provider is temporarily unavailable.", label)
	}

	if detail != "" {
		return fmt.Sprintf("%s request failed: %s", label, detail)
	}
	if e.StatusCode > 0 {
		return fmt.Sprintf("%s request failed with status %d.", label, e.StatusCode)
	}
	return fmt.Sprintf("%s request failed.", label)
}

func findAIRequestError(err error) (*AIRequestError, bool) {
	for err != nil {
		if requestErr, ok := err.(*AIRequestError); ok {
			return requestErr, true
		}
		type unwrapper interface{ Unwrap() error }
		next, ok := err.(unwrapper)
		if !ok {
			break
		}
		err = next.Unwrap()
	}
	return nil, false
}

func parseAIError(provider Provider, endpoint string, statusCode int, retryAfter string, body []byte) *AIRequestError {
	bodyText := truncateAIErrorDetail(strings.TrimSpace(string(body)))
	requestErr := &AIRequestError{
		Provider:     provider,
		Endpoint:     endpoint,
		StatusCode:   statusCode,
		RetryAfter:   strings.TrimSpace(retryAfter),
		ResponseBody: bodyText,
	}

	var payload struct {
		Error struct {
			Message  string `json:"message"`
			Code     any    `json:"code"`
			Metadata struct {
				ErrorType string `json:"error_type"`
			} `json:"metadata"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err == nil {
		requestErr.Message = strings.TrimSpace(payload.Error.Message)
		requestErr.ErrorType = strings.TrimSpace(payload.Error.Metadata.ErrorType)
	}
	if requestErr.Message == "" {
		requestErr.Message = bodyText
	}

	return requestErr
}

func providerLabel(provider Provider) string {
	switch provider {
	case ProviderOpenRouter:
		return "OpenRouter"
	case ProviderOllama:
		return "Ollama"
	default:
		if strings.TrimSpace(string(provider)) == "" {
			return "AI provider"
		}
		return string(provider)
	}
}

func truncateAIErrorDetail(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) <= maxAIErrorDetailLength {
		return value
	}
	return strings.TrimSpace(value[:maxAIErrorDetailLength]) + "..."
}

func retryAfterPhrase(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return "later"
	}
	if seconds == 1 {
		return "in 1 second"
	}
	if seconds < 60 {
		return fmt.Sprintf("in %d seconds", seconds)
	}
	minutes := (seconds + 59) / 60
	if minutes == 1 {
		return "in 1 minute"
	}
	return fmt.Sprintf("in %d minutes", minutes)
}
