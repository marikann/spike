//    \\ SPIKE: Secure your secrets with SPIFFE.
//  \\\\\ Copyright 2024-present SPIKE contributors.
// \\\\\\\ SPDX-License-Identifier: Apache-2.0

package persist

import (
	"context"

	"github.com/spiffe/spike-sdk-go/api/entity/data"

	"github.com/spiffe/spike/app/nexus/internal/env"
	"github.com/spiffe/spike/internal/log"
	"github.com/spiffe/spike/pkg/retry"
)

type retryHandler[T any] func() (T, error)

func doRetry[T any](ctx context.Context, handler retryHandler[T]) (T, error) {
	return retry.NewTypedRetrier[T](
		retry.NewExponentialRetrier(),
	).RetryWithBackoff(ctx, handler)
}

func StorePolicy(policy data.Policy) {
	const fName = "storePolicy"

	be := Backend()
	if be == nil {
		return // No cache available
	}

	if policy.Id == "" {
		return
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		env.DatabaseOperationTimeout(),
	)
	defer cancel()

	if err := be.StorePolicy(ctx, policy); err != nil {
		// Log error but continue - memory is source of truth
		log.Log().Warn(fName,
			"msg", "Failed to cache policy",
			"id", policy.Id,
			"err", err.Error(),
		)
	}
}

func DeletePolicy(id string) {
	const fName = "deletePolicy"

	be := Backend()
	if be == nil {
		return
	}

	if id == "" {
		return
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		env.DatabaseOperationTimeout(),
	)
	defer cancel()

	if err := be.DeletePolicy(ctx, id); err != nil {
		// Log error but continue - memory is source of truth
		log.Log().Warn(fName,
			"msg", "Failed to delete policy from cache",
			"id", id,
			"err", err.Error(),
		)
	}
}

func ReadPolicy(id string) *data.Policy {
	const fName = "readPolicy"

	be := Backend()
	if be == nil {
		return nil
	}

	if id == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(
		context.Background(), env.DatabaseOperationTimeout(),
	)
	defer cancel()

	cachedPolicy, err := doRetry(ctx, func() (*data.Policy, error) {
		return be.LoadPolicy(ctx, id)
	})

	if err != nil {
		log.Log().Warn(fName,
			"msg", "Failed to load policy from cache after retries",
			"id", id,
			"err", err.Error(),
		)
		return nil
	}

	return cachedPolicy
}
