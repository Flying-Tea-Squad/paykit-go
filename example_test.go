package paykit_test

import (
	"context"
	"fmt"

	paykit "github.com/Flying-Tea-Squad/paykit-go"
	"github.com/Flying-Tea-Squad/paykit-go/bogus"
)

// ExampleNewKES demonstrates creating and inspecting monetary values in Kenyan shillings.
func ExampleNewKES() {
	money := paykit.NewKES(150.50)

	fmt.Println("Amount in cents:", money.Amount())
	fmt.Println("Currency:", money.Currency())
	fmt.Println("Display string:", money.String())

	// Output:
	// Amount in cents: 15050
	// Currency: KES
	// Display string: KES 150.50
}

// ExampleFormatKenyanPhone demonstrates standardizing Kenyan mobile numbers into E.164 format.
func ExampleFormatKenyanPhone() {
	p1, _ := paykit.FormatKenyanPhone("0712345678")
	p2, _ := paykit.FormatKenyanPhone("+254712345678")
	p3, _ := paykit.FormatKenyanPhone("0700-000-001")

	fmt.Println("Local format:", p1)
	fmt.Println("International format:", p2)
	fmt.Println("Delimited format:", p3)

	// Output:
	// Local format: 254712345678
	// International format: 254712345678
	// Delimited format: 254700000001
}

// ExampleMapGatewayErrorCode demonstrates mapping provider-specific error codes
// to standardized PayKit error constants.
func ExampleMapGatewayErrorCode() {
	// Map known M-Pesa error codes
	errUserCancel := paykit.MapGatewayErrorCode("mpesa", "1032")
	errPinTimeout := paykit.MapGatewayErrorCode("mpesa", "1037")

	// Map generic/fallback error codes
	errInsufficient := paykit.MapGatewayErrorCode("unknown_provider", "INSUFFICIENT_FUNDS")

	// Unmapped error codes fall back to generic processing_error
	errUnknown := paykit.MapGatewayErrorCode("mpesa", "999999")

	fmt.Println("M-Pesa 1032:", errUserCancel)
	fmt.Println("M-Pesa 1037:", errPinTimeout)
	fmt.Println("Default fallback:", errInsufficient)
	fmt.Println("Unknown code:", errUnknown)

	// Output:
	// M-Pesa 1032: user_cancelled
	// M-Pesa 1037: pin_timeout
	// Default fallback: insufficient_funds
	// Unknown code: processing_error
}

// ExampleGateway demonstrates payment collection and capability discovery
// using the bogus test gateway double.
func ExampleGateway() {
	ctx := context.Background()

	// Instantiate the test double gateway
	var gw paykit.Gateway = bogus.New()

	// Normalize phone number
	phone, err := paykit.FormatKenyanPhone("0712345678")
	if err != nil {
		fmt.Printf("Phone formatting failed: %v\n", err)
		return
	}

	// Initiate a customer payment charge (KES 100.00 -> cents ending in 00 yields StatusSuccess)
	chargeResp, err := gw.Charge(ctx, &paykit.ChargeRequest{
		Amount:      paykit.NewKES(100.00),
		Phone:       phone,
		Description: "Order #104",
		Reference:   "ORD-104",
	})
	if err != nil {
		fmt.Printf("Charge transport error: %v\n", err)
		return
	}

	fmt.Println("Gateway Name:", gw.Name())
	fmt.Println("Charge Accepted:", chargeResp.Success)
	fmt.Println("Charge Status:", chargeResp.Status)

	// Discover optional capability: Disburser
	if disburser, ok := gw.(paykit.Disburser); ok {
		disbResp, err := disburser.Disburse(ctx, &paykit.DisbursementRequest{
			Amount:      paykit.NewKES(50.00),
			Phone:       phone,
			Description: "Payout #104",
			Reference:   "DISB-104",
		})
		if err == nil {
			fmt.Println("Disbursement Accepted:", disbResp.Success)
			fmt.Println("Disbursement Status:", disbResp.Status)
		}
	}

	// Output:
	// Gateway Name: bogus
	// Charge Accepted: true
	// Charge Status: success
	// Disbursement Accepted: true
	// Disbursement Status: success
}
