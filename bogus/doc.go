// Package bogus provides a zero-dependency, in-memory test double implementing
// [paykit.Gateway] and [paykit.Disburser] for deterministic automated testing.
//
// # Deterministic Amount Suffix Routing
//
// Rather than relying on external sandbox networks or complex mock servers,
// [Client] simulates various payment scenarios deterministically based on the last two digits
// (cents / minor units) of the requested amount:
//
//   - Cents ending in 00 ([SuffixSuccess]): simulates successful settlement ([paykit.StatusSuccess]).
//   - Cents ending in 02 ([SuffixPinTimeout]): simulates customer PIN entry timeout ([paykit.StatusTimeout]).
//   - Cents ending in 05 ([SuffixUserCancelled]): simulates customer prompt cancellation ([paykit.StatusCancelled]).
//   - Cents ending in 50 ([SuffixInsufficientFunds]): simulates insufficient account funds during disbursement ([paykit.StatusFailed]).
//
// Any other amount suffix defaults to [paykit.StatusProcessing].
//
// # Usage in Tests
//
// [Client] can be instantiated directly via [New] or fetched from the global registry
// using [paykit.Get]("bogus"):
//
//	client := bogus.New()
//
//	// Simulates a successful charge of KES 100.00 (suffix 00 -> StatusSuccess)
//	resp, err := client.Charge(ctx, &paykit.ChargeRequest{
//	    Amount: paykit.NewKES(100.00),
//	    Phone:  "254712345678",
//	})
//	// resp.Status == paykit.StatusSuccess
//
//	// Simulates a customer PIN timeout using KES 100.02 (suffix 02 -> StatusTimeout)
//	timeoutResp, err := client.Charge(ctx, &paykit.ChargeRequest{
//	    Amount: paykit.NewKES(100.02),
//	    Phone:  "254712345678",
//	})
//	// timeoutResp.Status == paykit.StatusTimeout
//
// # Concurrency
//
// [Client] is stateless and safe for concurrent use across multiple goroutines.
package bogus
