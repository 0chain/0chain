package storagesc

import (
	"errors"
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
	"0chain.net/core/util/entitywrapper"
	"github.com/0chain/common/core/logging"
)

//msgp:ignore StorageNode StorageAllocation AllocationChallenges storageNodeBase WriteMarker writeMarkerBase
//go:generate msgp -io=false -tests=false -unexported -v

func init() {
	entitywrapper.RegisterWrapper(&StorageNode{},
		map[string]entitywrapper.EntityI{
			entitywrapper.DefaultOriginVersion: &writeMarkerV1{},
			"v2":                               &storageNodeV2{},
		})
}

type WriteMarker struct {
	entitywrapper.Wrapper
}

func (wm *WriteMarker) TypeName() string {
	return "write_marker"
}

func (wm *WriteMarker) UnmarshalMsg(data []byte) ([]byte, error) {
	return wm.UnmarshalMsgType(data, wm.TypeName())
}

func (wm *WriteMarker) UnmarshalJSON(data []byte) error {
	return wm.UnmarshalJSONType(data, wm.TypeName())
}

func (wm *WriteMarker) Msgsize() (s int) {
	return wm.Entity().Msgsize()
}

func (wm *WriteMarker) GetVersion() string {
	return wm.Entity().GetVersion()
}

func (wm *WriteMarker) mustBase() *writeMarkerBase {
	b, ok := wm.Base().(*writeMarkerBase)
	if !ok {
		logging.Logger.Panic("invalid write marker base type")
	}
	return b
}

func (wm *WriteMarker) VerifySignature(
	clientPublicKey string,
	balances cstate.StateContextI,
) bool {
	var hashData, signature string
	switch wm.GetVersion() {
	case entitywrapper.DefaultOriginVersion:
		wm1 := wm.Entity().(*writeMarkerV1)
		hashData = wm1.GetHashData()
		signature = wm1.Signature
	case "v2":
		wm2 := wm.Entity().(*writeMarkerV2)
		hashData = wm2.GetHashData()
		signature = wm2.Signature
	}
	signatureHash := encryption.Hash(hashData)
	signatureScheme := balances.GetSignatureScheme()
	if err := signatureScheme.SetPublicKey(clientPublicKey); err != nil {
		return false
	}
	sigOK, err := signatureScheme.Verify(signature, signatureHash)
	if err != nil {
		return false
	}
	if !sigOK {
		return false
	}
	return true
}

func (wm *WriteMarker) Verify(allocRoot, prevRoot string) bool {
	wmb := wm.mustBase()
	if wmb.AllocationRoot != allocRoot || wmb.PreviousAllocationRoot != prevRoot {
		return false
	}
	if len(wmb.AllocationID) == 0 || len(wmb.BlobberID) == 0 ||
		len(wmb.ClientID) == 0 || wmb.Timestamp == 0 {
		return false
	}
	return true
}

type writeMarkerBase writeMarkerV1

func (wm *writeMarkerBase) CommitChangesTo(e entitywrapper.EntityI) {
}

type writeMarkerV1 struct {
	AllocationRoot         string           `json:"allocation_root"`
	PreviousAllocationRoot string           `json:"prev_allocation_root"`
	FileMetaRoot           string           `json:"file_meta_root"`
	AllocationID           string           `json:"allocation_id"`
	Size                   int64            `json:"size"`
	BlobberID              string           `json:"blobber_id"`
	Timestamp              common.Timestamp `json:"timestamp"`
	ClientID               string           `json:"client_id"`
	Signature              string           `json:"signature"`
}

func (wm1 *writeMarkerV1) GetVersion() string {
	return entitywrapper.DefaultOriginVersion
}

func (wm1 *writeMarkerV1) GetBase() entitywrapper.EntityBaseI {
	b := writeMarkerBase(*wm1)
	return &b
}

func (wm1 *writeMarkerV1) MigrateFrom(e entitywrapper.EntityI) error {
	// nothing to migrate as this is original version of the write marker
	return nil
}

func (wm1 *writeMarkerV1) GetHashData() string {
	hashData := fmt.Sprintf(
		"%s:%s:%s:%s:%s:%s:%d:%d",
		wm1.AllocationRoot, wm1.PreviousAllocationRoot,
		wm1.FileMetaRoot, wm1.AllocationID,
		wm1.BlobberID, wm1.ClientID, wm1.Size, wm1.Timestamp)
	return hashData
}

type writeMarkerV2 struct {
	Version                string           `json:"version"`
	AllocationRoot         string           `json:"allocation_root"`
	PreviousAllocationRoot string           `json:"prev_allocation_root"`
	FileMetaRoot           string           `json:"file_meta_root"`
	AllocationID           string           `json:"allocation_id"`
	Size                   int64            `json:"size"`
	ChainSize              int64            `json:"chain_size"`
	ChainHash              string           `json:"chain_hash"`
	BlobberID              string           `json:"blobber_id"`
	Timestamp              common.Timestamp `json:"timestamp"`
	ClientID               string           `json:"client_id"`
	Signature              string           `json:"signature"`
}

func (wm2 *writeMarkerV2) GetVersion() string {
	return "v2"
}

func (wm2 *writeMarkerV2) GetBase() entitywrapper.EntityBaseI {
	return &writeMarkerBase{
		AllocationRoot:         wm2.AllocationRoot,
		PreviousAllocationRoot: wm2.PreviousAllocationRoot,
		FileMetaRoot:           wm2.FileMetaRoot,
		AllocationID:           wm2.AllocationID,
		Size:                   wm2.Size,
		BlobberID:              wm2.BlobberID,
		Timestamp:              wm2.Timestamp,
		ClientID:               wm2.ClientID,
		Signature:              wm2.Signature,
	}
}

func (wm2 *writeMarkerV2) MigrateFrom(e entitywrapper.EntityI) error {
	v1, ok := e.(*writeMarkerV1)
	if !ok {
		return errors.New("struct migrate fail, wrong writemarker type")
	}
	wm2.ApplyBaseChanges(writeMarkerBase(*v1))
	wm2.Version = "v2"
	return nil
}

func (wm2 *writeMarkerV2) ApplyBaseChanges(wmb writeMarkerBase) {
	wm2.AllocationRoot = wmb.AllocationRoot
	wm2.PreviousAllocationRoot = wmb.PreviousAllocationRoot
	wm2.FileMetaRoot = wmb.FileMetaRoot
	wm2.AllocationID = wmb.AllocationID
	wm2.Size = wmb.Size
	wm2.BlobberID = wmb.BlobberID
	wm2.Timestamp = wmb.Timestamp
	wm2.ClientID = wmb.ClientID
	wm2.Signature = wmb.Signature
}

func (wm2 *writeMarkerV2) GetHashData() string {
	var hashData string
	if wm2.ChainHash != "" {
		hashData = fmt.Sprintf("%s:%s:%s:%s:%s:%s:%s:%d:%d:%d",
			wm2.AllocationRoot, wm2.PreviousAllocationRoot,
			wm2.FileMetaRoot, wm2.ChainHash, wm2.AllocationID, wm2.BlobberID,
			wm2.ClientID, wm2.Size, wm2.ChainSize, wm2.Timestamp)
	} else {
		hashData = fmt.Sprintf(
			"%s:%s:%s:%s:%s:%s:%d:%d",
			wm2.AllocationRoot, wm2.PreviousAllocationRoot,
			wm2.FileMetaRoot, wm2.AllocationID,
			wm2.BlobberID, wm2.ClientID, wm2.Size, wm2.Timestamp)
	}
	return hashData
}
