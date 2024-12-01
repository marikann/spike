//    \\ SPIKE: Secure your secrets with SPIFFE.
//  \\\\\ Copyright 2024-present SPIKE contributors.
// \\\\\\\ SPDX-License-Identifier: Apache-2.0

package secret

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	spike "github.com/spiffe/spike-sdk-go/api"
	"github.com/spiffe/spike-sdk-go/api/entity/data"
)

// newSecretUndeleteCommand creates and returns a new cobra.Command for restoring
// deleted secrets. It configures a command that allows users to restore one or
// more previously deleted versions of a secret at a specified path.
//
// Parameters:
//   - source: X.509 source for workload API authentication
//
// The command accepts a single argument:
//   - path: Location of the secret to restore
//
// Flags:
//   - --versions, -v (string): Comma-separated list of version numbers to
//     restore
//   - "0" or empty: Restores current version only (default)
//   - "1,2,3": Restores specific versions
//
// Returns:
//   - *cobra.Command: Configured undelete command
//
// Example Usage:
//
//	spike secret undelete secret/ella           # Restores current version
//	spike secret undelete secret/ella -v 1,2,3  # Restores specific versions
//	spike secret undelete secret/ella -v 0,1,2  # Restores current version plus 1,2
//
// The command performs trust to ensure:
//   - Exactly one path argument is provided
//   - Version numbers are valid non-negative integers
//   - Version strings are properly formatted
//
// Note: Command currently provides feedback about intended operations
// but actual restoration functionality is pending implementation
func newSecretUndeleteCommand(source *workloadapi.X509Source) *cobra.Command {
	var undeleteCmd = &cobra.Command{
		Use:   "undelete <path>",
		Short: "Undelete secrets at the specified path",
		Long: `Undelete secrets at the specified path. 
Specify versions using -v or --versions flag with comma-separated values.
Version 0 refers to the current/latest version.
If no version is specified, defaults to undeleting the current version.

Examples:
  spike secret undelete secret/ella           # Undeletes current version
  spike secret undelete secret/ella -v 1,2,3  # Undeletes specific versions
  spike secret undelete secret/ella -v 0,1,2  # Undeletes current version plus versions 1 and 2`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			state, err := spike.CheckInitState(source)
			if err != nil {
				fmt.Println("Failed to check initialization state:")
				fmt.Println(err.Error())
				return
			}

			if state == data.NotInitialized {
				fmt.Println("Please initialize SPIKE first by running 'spike init'.")
				return
			}

			path := args[0]
			versions, _ := cmd.Flags().GetString("versions")
			if versions == "" {
				versions = "0"
			}

			// Parse and validate versions
			versionList := strings.Split(versions, ",")
			for _, v := range versionList {
				version, err := strconv.Atoi(strings.TrimSpace(v))

				if err != nil {
					fmt.Printf("Error: invalid version number: %s\n", v)
					return
				}

				if version < 0 {
					fmt.Printf(
						"Error: version numbers cannot be negative: %s\n", v,
					)
					return
				}
			}

			err = spike.UndeleteSecret(source, path, versionList)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}

			fmt.Println("OK")
		},
	}

	undeleteCmd.Flags().StringP("versions", "v", "0",
		"Comma-separated list of versions to undelete")

	return undeleteCmd
}
