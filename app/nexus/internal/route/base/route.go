//    \\ SPIKE: Secure your secrets with SPIFFE.
//  \\\\\ Copyright 2024-present SPIKE contributors.
// \\\\\\\ SPDX-License-Identifier: Apache-2.0

// Package base contains the fundamental building blocks and core functions
// for handling HTTP requests in the Spike Nexus application. It provides
// the routing logic to map API actions and URL paths to their respective
// handlers while ensuring seamless request processing and response generation.
// This package serves as a central point for managing incoming API calls
// and delegating them to the correct functional units based on specified rules.
package base

import (
	"fmt"
	"github.com/spiffe/spike/app/nexus/internal/route/operator"
	"net/http"

	"github.com/spiffe/spike/app/nexus/internal/route/acl/policy"
	"github.com/spiffe/spike/app/nexus/internal/route/secret"
	"github.com/spiffe/spike/internal/log"
	"github.com/spiffe/spike/internal/net"
)

// Route handles all incoming HTTP requests by dynamically selecting and
// executing the appropriate handler based on the request path and HTTP method.
// It uses a factory function to create the specific handler for the given URL
// path and HTTP method combination.
//
// Parameters:
//   - w: The HTTP ResponseWriter to write the response to
//   - r: The HTTP Request containing the client's request details
func Route(
	w http.ResponseWriter, r *http.Request, a *log.AuditEntry,
) error {
	return net.RouteFactory[net.SpikeNexusApiAction](
		net.ApiUrl(r.URL.Path),
		net.SpikeNexusApiAction(r.URL.Query().Get(net.KeyApiAction)),
		r.Method,
		func(a net.SpikeNexusApiAction, p net.ApiUrl) net.Handler {
			switch {
			case a == net.ActionNexusDefault && p == net.SpikeNexusUrlSecrets:
				return secret.RoutePutSecret
			case a == net.ActionNexusGet && p == net.SpikeNexusUrlSecrets:
				return secret.RouteGetSecret
			case a == net.ActionNexusDelete && p == net.SpikeNexusUrlSecrets:
				return secret.RouteDeleteSecret
			case a == net.ActionNexusUndelete && p == net.SpikeNexusUrlSecrets:
				return secret.RouteUndeleteSecret
			case a == net.ActionNexusList && p == net.SpikeNexusUrlSecrets:
				return secret.RouteListPaths
			case a == net.ActionNexusDefault && p == net.SpikeNexusUrlPolicy:
				return policy.RoutePutPolicy
			case a == net.ActionNexusGet && p == net.SpikeNexusUrlPolicy:
				return policy.RouteGetPolicy
			case a == net.ActionNexusDelete && p == net.SpikeNexusUrlPolicy:
				return policy.RouteDeletePolicy
			case a == net.ActionNexusList && p == net.SpikeNexusUrlPolicy:
				return policy.RouteListPolicies
			case a == net.ActionNexusGet && p == net.SpikeNexusUrlSecretsMetadata:
				return secret.RouteGetSecretMetadata
			case a == net.ActionNexusDefault && p == net.SpikeNexusUrlOperatorRestore:
				fmt.Println("operator.RouteRestore")
				return operator.RouteRestore
			case a == net.ActionNexusDefault && p == net.SpikeNexusUrlOperatorRecover:
				fmt.Println("operator.RouteRecover")
				return operator.RouteRecover
			default:
				return net.Fallback
			}
		})(w, r, a)
}
