// Package cosmos defines and registers usql's Azure CosmosDB driver.
//
// See: https://github.com/btnguyen2k/gocosmos
package cosmos

import (
	_ "github.com/btnguyen2k/gocosmos" // DRIVER
	"github.com/rmasci/usql/drivers"
)

func init() {
	drivers.Register("cosmos", drivers.Driver{})
}
