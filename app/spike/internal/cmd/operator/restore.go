//    \\ SPIKE: Secure your secrets with SPIFFE.
//  \\\\\ Copyright 2024-present SPIKE contributors.
// \\\\\\\ SPDX-License-Identifier: Apache-2.0

package operator

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	spike "github.com/spiffe/spike-sdk-go/api"
	"golang.org/x/term"

	"github.com/spiffe/spike/app/spike/internal/trust"
	"github.com/spiffe/spike/internal/auth"
	"github.com/spiffe/spike/internal/log"
)

// newOperatorRestoreCommand creates a new cobra command for restoration
// operations on SPIKE Nexus.
//
// This function creates a command that allows privileged operators with the
// 'restore' role to restore SPIKE Nexus after a system failure. The command
// accepts recovery shards interactively and initiates the restoration process.
//
// Parameters:
//   - source *workloadapi.X509Source: The X.509 source for SPIFFE
//     authentication.
//   - spiffeId string: The SPIFFE ID of the caller for role-based access
//     control.
//
// Returns:
//   - *cobra.Command: A cobra command that implements the restoration
//     functionality.
//
// The command performs the following operations:
//   - Verifies the caller has the 'restore' role, aborting otherwise.
//   - Authenticates the restoration request.
//   - Prompts the user to enter a recovery shard (input is hidden for
//     security).
//   - Sends the shard to the SPIKE API to contribute to restoration.
//   - Reports the status of the restoration process to the user.
//
// The command will abort with a fatal error if:
//   - The caller lacks the required 'restore' role.
//   - There's an error reading the recovery shard from input.
//   - The API call to restore using the shard fails.
//   - No status is returned from the restoration attempt.
//
// If restoration is incomplete (more shards needed), the command displays the
// current count of collected shards and instructs the user to run the command
// again to provide additional shards.
func newOperatorRestoreCommand(
	source *workloadapi.X509Source, spiffeId string,
) *cobra.Command {
	var restoreCmd = &cobra.Command{
		Use:   "restore",
		Short: "Restore SPIKE Nexus (do this if SPIKE Nexus cannot auto-recover)",
		Run: func(cmd *cobra.Command, args []string) {
			if !auth.IsPilotRestore(spiffeId) {
				fmt.Println("")
				fmt.Println(
					"  You need to have a `restore` role to use this command.")
				fmt.Println(
					"  Please run `./hack/spire-server-entry-restore-register.sh`")
				fmt.Println("  with necessary privileges to assign this role.")
				fmt.Println("")
				log.FatalLn("Aborting.")
			}

			trust.AuthenticateRestore(spiffeId)

			fmt.Println("(your input will be hidden as you paste/type it)")
			fmt.Print("Enter recovery shard: ")
			shard, err := term.ReadPassword(syscall.Stdin)
			if err != nil {
				_, e := fmt.Fprintf(os.Stderr, "Error reading shard: %v\n", err)
				if e != nil {
					return
				}
				os.Exit(1)
			}

			api := spike.NewWithSource(source)

			var ss [32]byte

			// shard is in `spike:$id:$base64` format
			// split it by :

			shardParts := strings.SplitN(string(shard), ":", 3)
			if len(shardParts) != 3 {
				log.FatalLn("Invalid shard format. Expected format: `spike:$id:$secret`.")
			}

			index := shardParts[1]
			base64Data := shardParts[2]

			decodedShard, err := base64.StdEncoding.DecodeString(base64Data)
			// Security: reset shard immediately after use.
			for i := 0; i < len(shard); i++ {
				shard[i] = 0
			}

			if err != nil {
				log.FatalLn("Failed to decode recovery shard: ", err.Error())
			}

			if len(decodedShard) != 32 {
				// Security: reset decodedShard immediately after use.
				for i := 0; i < 32; i++ {
					decodedShard[i] = 0
				}

				log.FatalLn("Invalid recovery shard length: ", len(decodedShard))
			}

			for i := 0; i < 32; i++ {
				ss[i] = decodedShard[i]
			}

			// Security: reset decodedShard immediately after use.
			for i := 0; i < 32; i++ {
				decodedShard[i] = 0
			}

			// TODO: sanitize data before attempting restoration

			// TODO: sanitize data before attempting save too. (the decoded version should be 32 bytes)

			// TODO parse index from `spike::$index:` and send to the restore request.
			// TODO: handle error.
			ix, _ := strconv.Atoi(index)

			status, err := api.Restore(ix, &ss)

			// Security: reset ss immediately after recovery.
			for i := 0; i < 32; i++ {
				ss[i] = 0
			}

			if err != nil {
				log.FatalLn(err.Error())
			}

			if status == nil {
				log.FatalLn("Didn't get any status while trying to restore SPIKE.")
			}

			if status.Restored {
				fmt.Println("")
				fmt.Println("  SPIKE is now restored and ready to use.")
				fmt.Println(
					"  Please run `./hack/spire-server-entry-su-register.sh`")
				fmt.Println(
					"  with necessary privileges to start using SPIKE as a superuser.")
				fmt.Println("")
			} else {
				fmt.Println("")
				fmt.Println(" Shards collected: ", status.ShardsCollected)
				fmt.Println(" Shards remaining: ", status.ShardsRemaining)
				fmt.Println(
					" Please run `spike operator restore` " +
						"again to provide the remaining shards.")
				fmt.Println("")
			}
		},
	}

	return restoreCmd
}
