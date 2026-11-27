// Package driver provides side-effect imports that register all built-in payment
// provider drivers with the [paykit] registry.
//
// # Usage
//
// Importing this package with a blank identifier automatically registers all
// available provider drivers (such as "mpesa", "airtel", "pesapal", and "bogus")
// into the global [paykit] registry:
//
//	import (
//	    "github.com/Flying-Tea-Squad/paykit-go"
//	    _ "github.com/Flying-Tea-Squad/paykit-go/driver"
//	)
//
//	func main() {
//	    gw, err := paykit.Get("mpesa")
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	    // use gw
//	}
package driver
