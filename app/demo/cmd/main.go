//    \\ SPIKE: Secure your secrets with SPIFFE.
//  \\\\\ Copyright 2024-present SPIKE contributors.
// \\\\\\\ SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"

	spike "github.com/spiffe/spike-sdk-go/api"
)

func main() {
	api := spike.New() // Use the default Workload API Socket
	defer api.Close()  // Close the connection when done

	path := "/tenants/demo/db/creds"
	version := 0

	// Create a Secret
	err := api.PutSecret(path, map[string]string{
		"username": "SPIKE",
		"password": "SPIKE_Rocks",
	})
	if err != nil {
		fmt.Println("Error writing secret:", err.Error())
		return
	}

	// TODO: maybe GetSecret should not have a `version` parameter
	// and an override should be used instead for versioned secret gets.
	// as in api.GetSecretVersioned(path, version)

	// Read the Secret
	secret, err := api.GetSecret(path, version)
	if err != nil {
		fmt.Println("Error reading secret:", err.Error())
		return
	}

	if secret == nil {
		fmt.Println("Secret not found.")
		return
	}

	fmt.Println("Secret found:")

	data := secret.Data
	for k, v := range data {
		fmt.Printf("%s: %s\n", k, v)
	}
}
