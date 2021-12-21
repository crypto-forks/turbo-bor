package commands

import (
	"github.com/ledgerwatch/erigon-lib/kv"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/consensus/bor"
	"github.com/ledgerwatch/erigon/rpc"
)

// BorAPI Bor specific routines
type BorAPI interface {
	GetSnapshot(number *rpc.BlockNumber) (*Snapshot, error)
	// GetAuthor(number rpc.BlockNumber) (*common.Address, error)
	GetSnapshotAtHash(hash common.Hash) (*Snapshot, error)
	GetSigners(number *rpc.BlockNumber) ([]common.Address, error)
	GetSignersAtHash(hash common.Hash) ([]common.Address, error)
	GetCurrentProposer() (common.Address, error)
	GetCurrentValidators() ([]*bor.Validator, error)
	// GetRootHash(start uint64, end uint64) (string, error)
	Test() (string, error)
}

// BorImpl is implementation of the BorAPI interface
type BorImpl struct {
	*BaseAPI
	db kv.RoDB
}

// NewBorAPI returns BorImpl instance
func NewBorAPI(base *BaseAPI, db kv.RoDB) *BorImpl {
	return &BorImpl{
		BaseAPI: base,
		db:      db,
	}
}
