//    \\ SPIKE: Secure your secrets with SPIFFE.
//  \\\\\ Copyright 2024-present SPIKE contributors.
// \\\\\\\ SPDX-License-Identifier: Apache-2.0

package base

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/spiffe/spike-sdk-go/api/entity/data"

	"github.com/spiffe/spike/app/nexus/internal/state/persist"
	"github.com/spiffe/spike/internal/auth"
)

var (
	ErrPolicyNotFound = errors.New("policy not found")
	ErrPolicyExists   = errors.New("policy already exists")
	ErrInvalidPolicy  = errors.New("invalid policy")
)

// CheckAccess determines if a given SPIFFE Id has the required permissions for
// a specific path. It first checks if the Id belongs to SPIKE Pilot (which has
// unrestricted access), then evaluates against all defined policies. Policies
// are checked in order, with wildcard patterns evaluated first, followed by
// specific pattern matching using regular expressions.
//
// Parameters:
//   - spiffeId: The SPIFFE Id of the requestor
//   - path: The resource path being accessed
//   - wants: Slice of permissions being requested
//
// Returns:
//   - bool: true if access is granted, false otherwise
//
// The function grants access if any of these conditions are met:
//  1. The requestor is a pilot
//  2. A matching policy has the super permission
//  3. A matching policy contains all requested permissions
//
// A policy matches when:
//  1. It has wildcard patterns ("*") for both SPIFFE Id and path, or
//  2. Its SPIFFE Id pattern matches the requestor's Id and its path pattern
//     matches the requested path
func CheckAccess(
	spiffeId string, path string, wants []data.PolicyPermission,
) bool {
	if auth.IsPilot(spiffeId) {
		return true
	}

	policies := ListPolicies()
	for _, policy := range policies {
		// Check wildcard pattern first
		if policy.SpiffeIdPattern == "*" && policy.PathPattern == "*" {
			if hasAllPermissions(policy.Permissions, wants) {
				return true
			}
			continue
		}

		// Check specific patterns using pre-compiled regexes

		if policy.SpiffeIdPattern != "*" {
			if !policy.IdRegex.MatchString(spiffeId) {
				continue
			}
		}

		if policy.PathPattern != "*" {
			if !policy.PathRegex.MatchString(path) {
				continue
			}
		}

		if contains(policy.Permissions, data.PermissionSuper) {
			return true
		}

		if hasAllPermissions(policy.Permissions, wants) {
			return true
		}
	}

	return false
}

// CreatePolicy creates a new policy in the system after validating and
// preparing it. The function compiles regex patterns, generates a UUID, and
// sets creation timestamp before storing the policy.
//
// Parameters:
//   - policy: The policy to create. Must have a non-empty Name field.
//     SpiffeIdPattern and PathPattern can be "*" for wildcard matching,
//     or valid regular expressions.
//
// Returns:
//   - data.Policy: The created policy, including generated Id and timestamps
//   - error: ErrInvalidPolicy if policy name is empty, or regex compilation
//     errors for invalid patterns
//
// The function performs the following modifications to the input policy:
//   - Compiles and stores regex patterns for non-wildcard SpiffeIdPattern
//     and PathPattern
//   - Generates and sets a new UUID as the policy Id
//   - Sets CreatedAt to current time if not already set
func CreatePolicy(policy data.Policy) (data.Policy, error) {
	if policy.Name == "" {
		return data.Policy{}, ErrInvalidPolicy
	}

	var err error

	// Check for duplicate policy name
	policies.Range(func(key, value interface{}) bool {
		if value.(data.Policy).Name == policy.Name {
			err = ErrPolicyExists
			return false // stop the iteration
		}
		return true
	})
	if err != nil {
		return data.Policy{}, err
	}

	// Compile and validate patterns
	if policy.SpiffeIdPattern != "*" {
		idRegex, err := regexp.Compile(policy.SpiffeIdPattern)
		if err != nil {
			return data.Policy{},
				errors.Join(
					ErrInvalidPolicy,
					fmt.Errorf("%s: %v", "invalid spiffeid pattern", err),
				)
		}
		policy.IdRegex = idRegex
	}

	if policy.PathPattern != "*" {
		pathRegex, err := regexp.Compile(policy.PathPattern)
		if err != nil {
			return data.Policy{},
				errors.Join(
					ErrInvalidPolicy,
					fmt.Errorf("%s: %v", "invalid path pattern", err),
				)
		}
		policy.PathRegex = pathRegex
	}

	// Generate Id and set creation time
	policy.Id = uuid.New().String()
	if policy.CreatedAt.IsZero() {
		policy.CreatedAt = time.Now()
	}

	policies.Store(policy.Id, policy)
	persist.StorePolicy(policy)

	return policy, nil
}

// GetPolicy retrieves a policy by its Id from the policy store.
//
// Parameters:
//   - id: The unique identifier of the policy to retrieve
//
// Returns:
//   - data.Policy: The retrieved policy if found
//   - error: ErrPolicyNotFound if no policy exists with the given Id.
func GetPolicy(id string) (data.Policy, error) {
	if value, exists := policies.Load(id); exists {
		return value.(data.Policy), nil
	}

	// Try loading from cache
	cachedPolicy := persist.ReadPolicy(id)
	if cachedPolicy == nil {
		return data.Policy{}, ErrPolicyNotFound
	}

	// Store in memory for future use
	policies.Store(id, *cachedPolicy)
	return *cachedPolicy, nil
}

// DeletePolicy removes a policy from the system by its Id.
//
// Parameters:
//   - id: The unique identifier of the policy to delete
//
// Returns:
//   - error: ErrPolicyNotFound if no policy exists with the given Id,
//     nil if the deletion was successful
func DeletePolicy(id string) error {
	if _, exists := policies.Load(id); !exists {
		return ErrPolicyNotFound
	}

	policies.Delete(id)
	persist.DeletePolicy(id)

	return nil
}

// ListPolicies retrieves all policies from the policy store.
// It iterates through the concurrent map of policies and returns them as a slice.
//
// Returns:
//   - []data.Policy: A slice containing all existing policies. Returns an empty
//     slice if no policies exist. The order of policies in the returned slice
//     is non-deterministic due to the concurrent nature of the underlying store.
func ListPolicies() []data.Policy {
	var result []data.Policy

	policies.Range(func(key, value interface{}) bool {
		result = append(result, value.(data.Policy))
		return true
	})

	return result
}

// ListPoliciesByPath returns all policies that match a specific path pattern.
// It filters the policy store and returns only policies where PathPattern
// exactly matches the provided pattern string.
//
// Parameters:
//   - pathPattern: The exact path pattern to match against policies
//
// Returns:
//   - []data.Policy: A slice of policies with matching PathPattern. Returns an
//     empty slice if no policies match. The order of policies in the returned
//     slice is non-deterministic due to the concurrent nature of the underlying
//     store.
func ListPoliciesByPath(pathPattern string) []data.Policy {
	var result []data.Policy

	policies.Range(func(key, value interface{}) bool {
		policy := value.(data.Policy)
		if policy.PathPattern == pathPattern {
			result = append(result, policy)
		}
		return true
	})

	return result
}

// ListPoliciesBySpiffeId returns all policies that match a specific SPIFFE Id
// pattern. It filters the policy store and returns only policies where
// SpiffeIdPattern exactly matches the provided pattern string.
//
// Parameters:
//   - spiffeIdPattern: The exact SPIFFE Id pattern to match against policies
//
// Returns:
//   - []data.Policy: A slice of policies with matching SpiffeIdPattern. Returns
//     an empty slice if no policies match. The order of policies in the returned
//     slice is non-deterministic due to the concurrent nature of the underlying
//     store.
func ListPoliciesBySpiffeId(spiffeIdPattern string) []data.Policy {
	var result []data.Policy

	policies.Range(func(key, value interface{}) bool {
		policy := value.(data.Policy)
		if policy.SpiffeIdPattern == spiffeIdPattern {
			result = append(result, policy)
		}
		return true
	})

	return result
}
