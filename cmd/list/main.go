// list provides Chrome tab listing command
package list

import (
	"fmt"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/nathants/chrome/lib"
)

func init() {
	lib.Commands["list"] = list
	lib.Args["list"] = listArgs{}
}

type listArgs struct {
}

func (listArgs) Description() string {
	return `list - List Chrome tabs

Lists all open tabs in Chrome (external mode only).

Example:
  chrome list`
}

func list() {
	var args listArgs
	arg.MustParse(&args)

	err := lib.ListTabs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}