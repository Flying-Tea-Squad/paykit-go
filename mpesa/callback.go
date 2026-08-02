package mpesa

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// STKPushCallbackResult represents the final outcome Safaricom sends to an
// application's callback URL after processing an STK Push request.
type STKPushCallbackResult struct {
	MerchantRequestID string
	CheckoutRequestID string
	ResultCode        int
	ResultDesc        string
	Msisdn            string
	Amount            int
	TransactionID     string
	ReferenceData     map[string]any
}

type stkPushCallbackEnvelope struct {
	Body struct {
		STKCallback *stkPushCallback `json:"stkCallback"`
	} `json:"Body"`
}

type stkPushCallback struct {
	MerchantRequestID string                   `json:"MerchantRequestID"`
	CheckoutRequestID string                   `json:"CheckoutRequestID"`
	ResultCode        *int                     `json:"ResultCode"`
	ResultDesc        string                   `json:"ResultDesc"`
	CallbackMetadata  *stkPushCallbackMetadata `json:"CallbackMetadata"`
}

type stkPushCallbackMetadata struct {
	Items []stkPushCallbackItem `json:"Item"`
}

type stkPushCallbackItem struct {
	Name  string          `json:"Name"`
	Value json.RawMessage `json:"Value"`
}

// ParseSTKPushCallback parses a Safaricom STK Push callback payload.
func ParseSTKPushCallback(data []byte) (*STKPushCallbackResult, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, errors.New("mpesa: parse STK Push callback: empty payload")
	}

	var envelope stkPushCallbackEnvelope
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&envelope); err != nil {
		return nil, fmt.Errorf("mpesa: parse STK Push callback: invalid JSON: %w", err)
	}

	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("mpesa: parse STK Push callback: payload contains multiple JSON values")
		}

		return nil, fmt.Errorf("mpesa: parse STK Push callback: invalid trailing JSON: %w", err)
	}

	callback := envelope.Body.STKCallback
	if callback == nil {
		return nil, errors.New("mpesa: parse STK Push callback: missing Body.stkCallback")
	}
	if strings.TrimSpace(callback.MerchantRequestID) == "" {
		return nil, errors.New("mpesa: parse STK Push callback: missing MerchantRequestID")
	}
	if strings.TrimSpace(callback.CheckoutRequestID) == "" {
		return nil, errors.New("mpesa: parse STK Push callback: missing CheckoutRequestID")
	}
	if callback.ResultCode == nil {
		return nil, errors.New("mpesa: parse STK Push callback: missing ResultCode")
	}
	if strings.TrimSpace(callback.ResultDesc) == "" {
		return nil, errors.New("mpesa: parse STK Push callback: missing ResultDesc")
	}

	result := &STKPushCallbackResult{
		MerchantRequestID: callback.MerchantRequestID,
		CheckoutRequestID: callback.CheckoutRequestID,
		ResultCode:        *callback.ResultCode,
		ResultDesc:        callback.ResultDesc,
	}

	if err := populateSTKPushCallbackMetadata(callback, result); err != nil {
		return nil, fmt.Errorf("mpesa: parse STK Push callback: %w", err)
	}

	return result, nil
}

func populateSTKPushCallbackMetadata(callback *stkPushCallback, result *STKPushCallbackResult) error {
	if callback.CallbackMetadata == nil {
		if *callback.ResultCode == 0 {
			return errors.New("successful callback missing CallbackMetadata")
		}

		return nil
	}

	result.ReferenceData = make(map[string]any, len(callback.CallbackMetadata.Items))
	seen := make(map[string]struct{}, len(callback.CallbackMetadata.Items))
	var hasAmount, hasTransactionID, hasMsisdn bool

	for _, item := range callback.CallbackMetadata.Items {
		if strings.TrimSpace(item.Name) == "" {
			return errors.New("metadata item has an empty Name")
		}
		if _, exists := seen[item.Name]; exists {
			return fmt.Errorf("duplicate metadata item %q", item.Name)
		}
		seen[item.Name] = struct{}{}

		var referenceValue any
		if len(item.Value) == 0 {
			return fmt.Errorf("metadata item %q is missing Value", item.Name)
		}
		if err := json.Unmarshal(item.Value, &referenceValue); err != nil {
			return fmt.Errorf("metadata item %q has invalid Value: %w", item.Name, err)
		}
		result.ReferenceData[item.Name] = referenceValue

		switch item.Name {
		case "Amount":
			amount, err := parseSTKPushCallbackAmount(item.Value)
			if err != nil {
				return err
			}
			result.Amount = amount
			hasAmount = true
		case "MpesaReceiptNumber":
			transactionID, err := parseSTKPushCallbackString(item.Value, "MpesaReceiptNumber")
			if err != nil {
				return err
			}
			result.TransactionID = transactionID
			hasTransactionID = true
		case "PhoneNumber":
			msisdn, err := parseSTKPushCallbackPhoneNumber(item.Value)
			if err != nil {
				return err
			}
			result.Msisdn = msisdn
			hasMsisdn = true
		}
	}

	if *callback.ResultCode != 0 {
		return nil
	}
	if !hasAmount {
		return errors.New("successful callback missing metadata item \"Amount\"")
	}
	if !hasTransactionID {
		return errors.New("successful callback missing metadata item \"MpesaReceiptNumber\"")
	}
	if !hasMsisdn {
		return errors.New("successful callback missing metadata item \"PhoneNumber\"")
	}

	return nil
}

func parseSTKPushCallbackAmount(data json.RawMessage) (int, error) {
	var amount int
	if err := json.Unmarshal(data, &amount); err != nil {
		return 0, errors.New("metadata item \"Amount\" must be a whole number")
	}
	if amount <= 0 {
		return 0, errors.New("metadata item \"Amount\" must be greater than zero")
	}

	return amount, nil
}

func parseSTKPushCallbackString(data json.RawMessage, name string) (string, error) {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return "", fmt.Errorf("metadata item %q must be a string", name)
	}
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("metadata item %q must not be empty", name)
	}

	return value, nil
}

func parseSTKPushCallbackPhoneNumber(data json.RawMessage) (string, error) {
	var phoneNumber string
	if err := json.Unmarshal(data, &phoneNumber); err == nil {
		if strings.TrimSpace(phoneNumber) == "" {
			return "", errors.New("metadata item \"PhoneNumber\" must not be empty")
		}

		return phoneNumber, nil
	}

	var number json.Number
	if err := json.Unmarshal(data, &number); err != nil {
		return "", errors.New("metadata item \"PhoneNumber\" must be a number or string")
	}
	if _, err := strconv.ParseUint(number.String(), 10, 64); err != nil || number.String() == "0" {
		return "", errors.New("metadata item \"PhoneNumber\" must be a positive whole number or non-empty string")
	}

	return number.String(), nil
}
