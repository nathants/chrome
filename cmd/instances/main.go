// instances provides command to list running Chrome instances
package instances

import (
	"fmt"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/nathants/chrome/lib"
)

func init() {
	lib.Commands["instances"] = listInstances
	lib.Args["instances"] = instancesArgs{}
}

type instancesArgs struct{}

func (instancesArgs) Description() string {
	return `instances - List running Chrome instances

Shows all Chrome instances launched with 'chrome launch' that are still running.
Each instance has a port and user-data-dir for persistent cookies/auth.

Example:
  chrome instances

Output:
  PORT   USER_DATA_DIR             STARTED
  9222   ~/.chrome                 2024-01-15T10:30:00Z
  9223   ~/.chrome-twitter         2024-01-15T11:00:00Z`
}

func listInstances() {
	var args instancesArgs
	arg.MustParse(&args)

	instances, err := lib.ListInstances()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if len(instances) == 0 {
		fmt.Println("No Chrome instances running")
		fmt.Println("")
		fmt.Println("Launch one with:")
		fmt.Println("  chrome launch")
		fmt.Println("  chrome launch --port 9223 --user-data-dir ~/.chrome-twitter")
		return
	}

	// Print header
	fmt.Printf("%-6s  %-40s  %s\n", "PORT", "USER_DATA_DIR", "STARTED")
	fmt.Printf("%-6s  %-40s  %s\n", "----", "-------------", "-------")

	for _, inst := range instances {
		userDataDir := inst.UserDataDir
		if len(userDataDir) > 40 {
			userDataDir = "..." + userDataDir[len(userDataDir)-37:]
		}
		fmt.Printf("%-6d  %-40s  %s\n", inst.Port, userDataDir, inst.StartedAt)
	}
}
