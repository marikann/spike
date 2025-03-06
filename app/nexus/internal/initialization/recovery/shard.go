//    \\ SPIKE: Secure your secrets with SPIFFE.
//  \\\\\ Copyright 2024-present SPIKE contributors.
// \\\\\\\ SPDX-License-Identifier: Apache-2.0

package recovery

import (
	"encoding/base64"
	"encoding/json"
	"net/url"

	"github.com/cloudflare/circl/secretsharing"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"github.com/spiffe/spike-sdk-go/api/entity/v1/reqres"
	apiUrl "github.com/spiffe/spike-sdk-go/api/url"
	network "github.com/spiffe/spike-sdk-go/net"

	"github.com/spiffe/spike/internal/auth"
	"github.com/spiffe/spike/internal/log"
	"github.com/spiffe/spike/internal/net"
)

func shardUrl(keeperApiRoot string) string {
	const fName = "shardUrl"

	u, err := url.JoinPath(keeperApiRoot, string(apiUrl.SpikeKeeperUrlShard))
	if err != nil {
		log.Log().Warn(
			fName, "msg", "Failed to join path", "url", keeperApiRoot,
		)
		return ""
	}
	return u
}

func shardResponse(
	source *workloadapi.X509Source, u string, keeperId string,
) []byte {
	const fName = "shardResponse"

	shardRequest := reqres.ShardRequest{}
	md, err := json.Marshal(shardRequest)
	if err != nil {
		log.Log().Warn(fName,
			"msg", "Failed to marshal request",
			"err", err, "keeper_id", keeperId)
		return []byte{}
	}

	client, err := network.CreateMtlsClientWithPredicate(
		source, auth.IsKeeper,
	)

	if err != nil {
		log.Log().Warn(fName,
			"msg", "Failed to create mTLS client",
			"err", err)
		return []byte{}
	}

	data, err := net.Post(client, u, md)
	if err != nil {
		log.Log().Warn(fName,
			"msg", "Failed to post",
			"err", err, "keeper_id", keeperId)
	}

	if len(data) == 0 {
		log.Log().Info(fName, "msg", "No data")
		return []byte{}
	}

	return data
}

func unmarshalShardResponse(data []byte) *reqres.ShardResponse {
	const fName = "unmarshalShardResponse"

	var res reqres.ShardResponse
	err := json.Unmarshal(data, &res)
	if err != nil {
		log.Log().Info(fName, "msg",
			"Failed to unmarshal response", "err", err)
		return nil
	}
	return &res
}

func shardContributionResponse(
	keeperId string, keepers map[string]string, u string,
	rootShares []secretsharing.Share, source *workloadapi.X509Source,
) []byte {
	const fName = "shardContributionResponse"

	client, err := network.CreateMtlsClientWithPredicate(source, auth.IsKeeper)

	if err != nil {
		log.Log().Warn(fName,
			"msg", "Failed to create mTLS client",
			"err", err)
		return []byte{}
	}

	// Do not zero-out `share`; rootShares is zeroed out elsewhere.
	share := findShare(keeperId, keepers, rootShares)

	contribution, err := share.Value.MarshalBinary()
	// Security: Ensure that the share is zeroed out before the function returns.
	defer func() {
		for i := range contribution {
			contribution[i] = 0
		}
	}()

	if err != nil {
		log.Log().Warn(fName,
			"msg", "Failed to marshal share",
			"err", err, "keeper_id", keeperId)
		return []byte{}
	}

	// TODO: ShardContributionRequest should use a [32]byte instead of a string
	// for better memory security.
	scr := reqres.ShardContributionRequest{
		KeeperId: keeperId,
		Shard:    base64.StdEncoding.EncodeToString(contribution),
	}
	defer func() {
		// TODO: do a proper cleanup instead.
		scr.Shard = ""
	}()

	md, err := json.Marshal(scr)
	if err != nil {
		log.Log().Warn(fName,
			"msg", "Failed to marshal request",
			"err", err, "keeper_id", keeperId)
		return []byte{}
	}

	data, err := net.Post(client, u, md)
	if err != nil {
		log.Log().Warn(fName, "msg",
			"Failed to post",
			"err", err, "keeper_id", keeperId)
	}
	// Security: Ensure that the md is zeroed out before the function exits.
	defer func() {
		for i := range md {
			md[i] = 0
		}
	}()

	if len(data) == 0 {
		log.Log().Info(fName, "msg", "No data")
		return []byte{}
	}

	return data
}
