//    \\ SPIKE: Secure your secrets with SPIFFE.
//  \\\\\ Copyright 2024-present SPIKE contributors.
// \\\\\\\ SPDX-License-Identifier: Apache-2.0

package secret

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	spike "github.com/spiffe/spike-sdk-go/api"

	"github.com/spiffe/spike/app/spike/internal/stdout"
	"github.com/spiffe/spike/app/spike/internal/trust"
)

// newSecretGetCommand creates and returns a new cobra.Command for retrieving
// secrets. It configures a command that fetches and displays secret data from a
// specified path.
//
// Parameters:
//   - source: X.509 source for workload API authentication
//
// The command accepts a single argument:
//   - path: Location of the secret to retrieve
//
// Flags:
//   - --version, -v (int): Specific version of the secret to retrieve
//     (default 0) where 0 represents the current version
//
// Returns:
//   - *cobra.Command: Configured get command
//
// The command will:
//  1. Verify SPIKE initialization status via admin token
//  2. Retrieve the secret from the specified path and version
//  3. Display all key-value pairs in the secret's data field
//
// Error cases:
//   - SPIKE not initialized: Prompts user to run 'spike init'
//   - Secret not found: Displays appropriate message
//   - Read errors: Displays error message
func newSecretGetCommand(
	source *workloadapi.X509Source, spiffeId string,
) *cobra.Command {
	var getCmd = &cobra.Command{
		Use:   "get <path>",
		Short: "Get secrets from the specified path",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			trust.Authenticate(spiffeId)

			api := spike.NewWithSource(source)

			path := args[0]
			version, _ := cmd.Flags().GetInt("version")

			if !validSecretPath(path) {
				fmt.Printf("Error: invalid secret path: %s\n", path)
				return
			}

			secret, err := api.GetSecretVersion(path, version)
			if err != nil {
				if err.Error() == "not ready" {
					stdout.PrintNotReady()
					return
				}

				fmt.Println("Error reading secret:", err.Error())
				return
			}

			if secret == nil {
				fmt.Println("Secret not found.")
				return
			}

			d := secret.Data
			for k, v := range d {
				fmt.Printf("%s: %s\n", k, v)
			}
		},
	}

	getCmd.Flags().IntP("version", "v", 0, "Specific version to retrieve")

	return getCmd
}
