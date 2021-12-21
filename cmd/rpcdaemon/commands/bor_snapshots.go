package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"github.com/ledgerwatch/erigon-lib/kv"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/consensus/bor"
	"github.com/ledgerwatch/erigon/core/rawdb"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/params"
	"github.com/ledgerwatch/erigon/rpc"
)

type Snapshot struct {
	config *params.BorConfig // Consensus engine parameters to fine tune behavior

	Number       uint64                    `json:"number"`       // Block number where the snapshot was created
	Hash         common.Hash               `json:"hash"`         // Block hash where the snapshot was created
	ValidatorSet *ValidatorSet             `json:"validatorSet"` // Validator set at this moment
	Recents      map[uint64]common.Address `json:"recents"`      // Set of recent signers for spam protections
}

type ValidatorSet struct {
	// NOTE: persisted via reflect, must be exported.
	Validators []*bor.Validator `json:"validators"`
	Proposer   *bor.Validator   `json:"proposer"`

	// cached (unexported)
	totalVotingPower int64
}

// GetSnapshot retrieves the state snapshot at a given block.
func (api *BorImpl) GetSnapshot(number *rpc.BlockNumber) (*Snapshot, error) {
	ctx := context.Background()
	tx, err := api.db.BeginRo(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Retrieve the requested block number (or current if none requested)
	var header *types.Header
	if number == nil || *number == rpc.LatestBlockNumber {
		header = rawdb.ReadCurrentHeader(tx)
	} else {
		header, _ = getHeaderByNumber(ctx, *number, api, tx)
	}
	// Ensure we have an actually valid block and return its snapshot
	if header == nil {
		return nil, errors.New("unknown block")
	}
	return snapshot(api, tx, header.Number.Uint64(), header.Hash())
}

func (api *BorImpl) Test() (string, error) {
	return "Hello World", nil
}

// helper functions
// getHeaderByNumber returns a block's header given a block number ignoring the block's transaction and uncle list (may be faster).
// derived from erigon_getHeaderByNumber implementation (see ./erigon_block.go)
func getHeaderByNumber(ctx context.Context, number rpc.BlockNumber, api *BorImpl, tx kv.Tx) (*types.Header, error) {
	// Pending block is only known by the miner
	if number == rpc.PendingBlockNumber {
		block := api.pendingBlock()
		if block == nil {
			return nil, nil
		}
		return block.Header(), nil
	}

	blockNum, err := getBlockNumber(number, tx)
	if err != nil {
		return nil, err
	}

	header := rawdb.ReadHeaderByNumber(tx, blockNum)
	if header == nil {
		return nil, fmt.Errorf("block header not found: %d", blockNum)
	}

	return header, nil
}

// snapshot retrieves the authorization snapshot at a given point in time.
func snapshot(api *BorImpl, tx kv.Tx, number uint64, hash common.Hash) (*Snapshot, error) {
	var snap *Snapshot
	// load on-disk checkpoints
	if s, err := loadSnapshot(api, tx, hash); err == nil {
		snap = s
	}

	if snap == nil {
		return nil, fmt.Errorf("unknown error while retrieving snapshot at block number %v", number)
	}

	return snap, nil
}

// loadSnapshot loads an existing snapshot from the database.
func loadSnapshot(api *BorImpl, tx kv.Tx, hash common.Hash) (*Snapshot, error) {
	blob, err := tx.GetOne(kv.CliqueSeparate, append([]byte("bor-"), hash[:]...))
	if err != nil {
		return nil, err
	}
	snap := new(Snapshot)
	if err := json.Unmarshal(blob, snap); err != nil {
		return nil, err
	}
	config, _ := api.BaseAPI.chainConfig(tx)
	snap.config = config.Bor

	// update total voting power
	if err := updateTotalVotingPower(snap.ValidatorSet); err != nil {
		return nil, err
	}

	return snap, nil
}

// Force recalculation of the set's total voting power.
func updateTotalVotingPower(vals *ValidatorSet) error {

	sum := int64(0)
	for _, val := range vals.Validators {
		// mind overflow
		sum = safeAddClip(sum, val.VotingPower)
		if sum > bor.MaxTotalVotingPower {
			return &bor.TotalVotingPowerExceededError{sum, vals.Validators}
		}
	}
	vals.totalVotingPower = sum
	return nil
}

// safe addition
func safeAdd(a, b int64) (int64, bool) {
	if b > 0 && a > math.MaxInt64-b {
		return -1, true
	} else if b < 0 && a < math.MinInt64-b {
		return -1, true
	}
	return a + b, false
}

func safeAddClip(a, b int64) int64 {
	c, overflow := safeAdd(a, b)
	if overflow {
		if b < 0 {
			return math.MinInt64
		}
		return math.MaxInt64
	}
	return c
}
