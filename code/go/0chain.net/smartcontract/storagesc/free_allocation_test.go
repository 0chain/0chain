package storagesc

import (
	"0chain.net/chaincore/chain"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/mocks"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/util"
	"encoding/hex"
	"encoding/json"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
	"time"
)

func TestAddFreeStorageAssigner(t *testing.T) {
	const (
		mockCooperationId            = "mock cooperation id"
		mockPublicKey                = "mock public key"
		mockAnnualTokenLimit         = 100
		mockExistingAnnualTokenLimit = 50
		mockNotOwner                 = "mock not owner"
	)
	var mockExistingRedeemed = []freeStorageRedeemed{
		{12, 20000, 10000}, {40, 50000, 10000},
	}

	type args struct {
		ssc      *StorageSmartContract
		txn      *transaction.Transaction
		input    []byte
		balances cstate.StateContextI
	}
	type want struct {
		err, getConfig, getExisting, saveNew bool
		errMsg                               string
	}
	type parameters struct {
		clientId string
		info     newFreeStorageAssignerInfo
		exists   bool
	}

	setExpectations := func(t *testing.T, name string, p parameters, want want) args {
		var balances = &mocks.StateContextI{}
		var txn = &transaction.Transaction{
			ClientID: p.clientId,
		}
		var ssc = &StorageSmartContract{
			SmartContract: sci.NewSC(ADDRESS),
		}
		var conf = &scConfig{
			MaxAnnualFreeAllocation: zcnToBalance(mockAnnualTokenLimit),
		}
		input, err := json.Marshal(p.info)
		require.NoError(t, err)

		if !want.getConfig {
			return args{ssc, txn, input, balances}
		}
		balances.On("GetTrieNode", scConfigKey(ssc.ID)).Return(conf, nil).Once()

		if !want.getExisting {
			return args{ssc, txn, input, balances}
		}
		var newRedeemed []freeStorageRedeemed
		if p.exists {
			newRedeemed = mockExistingRedeemed
			balances.On("GetTrieNode", freeStorageAssignerKey(ssc.ID, p.info.ClientId)).Return(&freeStorageAssigner{
				ClientId:             p.info.ClientId,
				PublicKey:            p.info.PublicKey,
				AnnualLimit:          mockExistingAnnualTokenLimit,
				FreeStoragesRedeemed: mockExistingRedeemed,
			}, nil).Once()
		} else {
			balances.On("GetTrieNode", freeStorageAssignerKey(ssc.ID, p.info.ClientId)).Return(nil, util.ErrValueNotPresent).Once()
		}

		if !want.saveNew {
			return args{ssc, txn, input, balances}
		}
		balances.On("InsertTrieNode", (freeStorageAssignerKey(ssc.ID, p.info.ClientId)),
			&freeStorageAssigner{
				ClientId:             p.info.ClientId,
				PublicKey:            p.info.PublicKey,
				AnnualLimit:          state.Balance(p.info.AnnualTokenLimit * floatToBalance),
				FreeStoragesRedeemed: newRedeemed,
			}).Return("", nil).Once()

		return args{ssc, txn, input, balances}
	}

	testCases := []struct {
		name       string
		parameters parameters
		want       want
	}{
		{
			name: "ok_new",
			parameters: parameters{
				clientId: owner,
				info: newFreeStorageAssignerInfo{
					ClientId:         mockCooperationId + "ok_new",
					PublicKey:        mockPublicKey,
					AnnualTokenLimit: mockAnnualTokenLimit,
				},
				exists: false,
			},
			want: want{false, true, true, true, ""},
		},
		{
			name: "ok_existing",
			parameters: parameters{
				clientId: owner,
				info: newFreeStorageAssignerInfo{
					ClientId:         mockCooperationId + "ok_new",
					PublicKey:        mockPublicKey,
					AnnualTokenLimit: mockAnnualTokenLimit,
				},
				exists: true,
			},
			want: want{false, true, true, true, ""},
		},
		{
			name: "not_owner",
			parameters: parameters{
				clientId: mockNotOwner,
				info: newFreeStorageAssignerInfo{
					ClientId:         mockCooperationId + "ok_new",
					PublicKey:        mockPublicKey,
					AnnualTokenLimit: mockAnnualTokenLimit,
				},

				exists: true,
			},
			want: want{
				true, false, false, false,
				"add_free_storage_assigner: unauthorized access - only the owner can update the variables",
			},
		},
	}
	for _, test := range testCases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			args := setExpectations(t, test.name, test.parameters, test.want)

			err := args.ssc.addFreeStorageAssigner(args.txn, args.input, args.balances)

			require.True(t, mock.AssertExpectationsForObjects(t, args.balances))
			require.EqualValues(t, test.want.err, err != nil)
			if err != nil {
				require.EqualValues(t, test.want.errMsg, err.Error())
			}
		})
	}
}

func TestFreeAllocationRequest(t *testing.T) {
	t.Skip()
	const (
		mockCooperationId = "mock cooperation id"
		//mockPublicKey                = "mock cooperation public key"

		mockExistingAnnualTokenLimit = 50
		mockNotOwner                 = "mock not owner"
		mockNumBlobbers              = 10
		mockRecipient                = "mock recipient"
		mockFreeTokens               = 5
		mockTimestamp                = 7000
		mockUserPublicKey            = "mock user public key"
		mockTransactionHash          = "12345678"
	)
	var mockMaxAnnualFreeAllocation = zcnToBalance(100354)
	var mockAnnualTokenLimit = zcnToBalance(100)
	var mockFreeAllocationSettings = freeAllocationSettings{
		DataShards:                 5,
		ParityShards:               5,
		Size:                       123456,
		ReadPriceRange:             PriceRange{0, 5000},
		WritePriceRange:            PriceRange{0, 5000},
		MaxChallengeCompletionTime: 1 * time.Hour,
		Duration:                   24 * 365 * time.Hour,
	}
	var mockAllBlobbers = &StorageNodes{}
	var conf = &scConfig{
		MinAllocSize:               1027,
		MinAllocDuration:           5 * time.Minute,
		MaxChallengeCompletionTime: 1 * time.Hour,
		MaxAnnualFreeAllocation:    mockMaxAnnualFreeAllocation,
		FreeAllocationSettings:     mockFreeAllocationSettings,
	}
	var now = common.Timestamp(10000)
	var mockChallengeCompletionTime = conf.MaxChallengeCompletionTime
	for i := 0; i < mockNumBlobbers; i++ {
		mockBlobber := &StorageNode{
			ID:       strconv.Itoa(i),
			Capacity: 536870912,
			Used:     73,
			Terms: Terms{
				MaxOfferDuration:        mockFreeAllocationSettings.Duration * 2,
				ReadPrice:               mockFreeAllocationSettings.ReadPriceRange.Max,
				ChallengeCompletionTime: mockChallengeCompletionTime,
			},
			LastHealthCheck: now - blobberHealthTime + 1,
		}
		mockAllBlobbers.Nodes.add(mockBlobber)
	}

	type args struct {
		ssc      *StorageSmartContract
		txn      *transaction.Transaction
		input    []byte
		balances cstate.StateContextI
	}
	type want struct {
		err    bool
		errMsg string
	}
	type parameters struct {
		assigner freeStorageAssigner
		marker   freeStorageMarker
		exists   bool
	}

	setExpectations := func(t *testing.T, name string, p parameters, want want) args {
		var balances = &mocks.StateContextI{}
		var txn = &transaction.Transaction{
			ClientID:     p.marker.Recipient,
			PublicKey:    mockUserPublicKey,
			CreationDate: now,
			Value:        zcnToInt64(p.marker.FreeTokens),
		}
		txn.Hash = mockTransactionHash
		var ssc = &StorageSmartContract{
			SmartContract: sci.NewSC(ADDRESS),
		}

		p.marker.Signature, p.assigner.PublicKey = signFreeAllocationMarker(t, p.marker)
		input, err := json.Marshal(p.marker)
		require.NoError(t, err)
		balances.On(
			"GetTrieNode",
			freeStorageAssignerKey(ssc.ID, p.marker.Giver),
		).Return(&p.assigner, nil).Once()

		balances.On("GetTrieNode", scConfigKey(ssc.ID)).Return(conf, nil).Once()

		balances.On("GetTrieNode", ALL_BLOBBERS_KEY).Return(
			mockAllBlobbers, nil,
		).Once()

		for _, blobber := range mockAllBlobbers.Nodes {
			balances.On(
				"GetTrieNode", stakePoolKey(ssc.ID, blobber.ID),
			).Return(newStakePool(), nil).Twice()
			balances.On(
				"InsertTrieNode", blobber.GetKey(ssc.ID), mock.Anything,
			).Return("", nil).Once()
			balances.On(
				"InsertTrieNode", stakePoolKey(ssc.ID, blobber.ID), mock.Anything,
			).Return("", nil).Once()
		}

		balances.On(
			"InsertTrieNode", ALL_BLOBBERS_KEY, mock.Anything,
		).Return("", nil).Once()
		balances.On(
			"GetTrieNode", writePoolKey(ssc.ID, p.marker.Recipient),
		).Return(nil, util.ErrValueNotPresent).Once()

		balances.On(
			"GetTrieNode", challengePoolKey(ssc.ID, txn.Hash),
		).Return(nil, util.ErrValueNotPresent).Once()
		balances.On(
			"InsertTrieNode", challengePoolKey(ssc.ID, txn.Hash), mock.Anything,
		).Return("", nil).Once()

		var clientAlloc = ClientAllocation{ClientID: p.marker.Recipient}
		balances.On(
			"GetTrieNode", clientAlloc.GetKey(ssc.ID),
		).Return(nil, util.ErrValueNotPresent).Once()
		balances.On(
			"GetTrieNode", ALL_ALLOCATIONS_KEY,
		).Return(&Allocations{}, nil).Once()

		allocation := StorageAllocation{ID: txn.Hash}
		balances.On(
			"GetTrieNode", allocation.GetKey(ssc.ID),
		).Return(nil, util.ErrValueNotPresent).Once()
		balances.On(
			"InsertTrieNode", ALL_ALLOCATIONS_KEY, mock.Anything,
		).Return("", nil).Once()
		balances.On(
			"InsertTrieNode", clientAlloc.GetKey(ssc.ID), mock.Anything,
		).Return("", nil).Once()
		balances.On(
			"InsertTrieNode", allocation.GetKey(ssc.ID), mock.Anything,
		).Return("", nil).Once()

		balances.On(
			"InsertTrieNode",
			freeStorageAssignerKey(ssc.ID, p.marker.Giver),
			&freeStorageAssigner{
				ClientId:    p.assigner.ClientId,
				PublicKey:   p.assigner.PublicKey,
				AnnualLimit: p.assigner.AnnualLimit,
				FreeStoragesRedeemed: append(
					p.assigner.FreeStoragesRedeemed,
					freeStorageRedeemed{
						Amount:    zcnToBalance(p.marker.FreeTokens),
						When:      txn.CreationDate,
						Timestamp: p.marker.Timestamp,
					}),
			},
		).Return("", nil).Once()

		balances.On("AddMint", &state.Mint{
			Minter:     ADDRESS,
			ToClientID: ADDRESS,
			Amount:     zcnToBalance(p.marker.FreeTokens),
		}).Return(nil).Once()

		balances.On("InsertTrieNode",
			writePoolKey(ssc.ID, p.marker.Recipient),
			mock.MatchedBy(func(wp *writePool) bool {
				pool, found := wp.Pools.get(mockTransactionHash)
				require.True(t, found)
				return pool.Balance == zcnToBalance(p.marker.FreeTokens) &&
					pool.ID == mockTransactionHash &&
					pool.AllocationID == mockTransactionHash &&
					len(pool.Blobbers) == mockNumBlobbers &&
					pool.ExpireAt == common.Timestamp(common.ToTime(txn.CreationDate).Add(
						conf.FreeAllocationSettings.Duration).Unix())+toSeconds(mockChallengeCompletionTime)
			})).Return("", nil).Once()

		return args{ssc, txn, input, balances}
	}

	testCases := []struct {
		name       string
		parameters parameters
		want       want
	}{
		{
			name: "ok_no_previous",
			parameters: parameters{
				marker: freeStorageMarker{
					Giver:      mockCooperationId + "ok",
					Recipient:  mockRecipient,
					FreeTokens: mockFreeTokens,
					Timestamp:  mockTimestamp,
				},
				assigner: freeStorageAssigner{
					ClientId:    mockCooperationId + "ok",
					AnnualLimit: mockAnnualTokenLimit,
				},
			},
			want: want{false, ""},
		},
		{
			name: "ok",
			parameters: parameters{
				marker: freeStorageMarker{
					Giver:      mockCooperationId + "ok",
					Recipient:  mockRecipient,
					FreeTokens: mockFreeTokens,
					Timestamp:  mockTimestamp,
				},
				assigner: freeStorageAssigner{
					ClientId:    mockCooperationId + "ok",
					PublicKey:   "",
					AnnualLimit: mockAnnualTokenLimit,
					FreeStoragesRedeemed: []freeStorageRedeemed{
						{},
					},
				},
			},
			want: want{false, ""},
		},
	}
	for _, test := range testCases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			args := setExpectations(t, test.name, test.parameters, test.want)

			resp, err := args.ssc.freeAllocationRequest(args.txn, args.input, args.balances)

			require.True(t, mock.AssertExpectationsForObjects(t, args.balances))
			require.EqualValues(t, test.want.err, err != nil)
			if err != nil {
				require.EqualValues(t, test.want.errMsg, err.Error())
				return
			}
			require.True(t, mock.AssertExpectationsForObjects(t, args.balances))
			resp = resp
		})
	}
}

func TestUpdateFreeStorageRequest(t *testing.T) {
	const (
		mockCooperationId            = "mock cooperation id"
		mockAllocationId             = "mock allocation id"
		mockExistingAnnualTokenLimit = 50
		mockNotOwner                 = "mock not owner"
		mockNumBlobbers              = 10
		mockRecipient                = "mock recipient"
		mockFreeTokens               = 5
		mockTimestamp                = 7000
		mockUserPublicKey            = "mock user public key"
		mockTransactionHash          = "12345678"
	)
	var mockMaxAnnualFreeAllocation = zcnToBalance(100354)
	var mockAnnualTokenLimit = zcnToBalance(100)
	var mockFreeAllocationSettings = freeAllocationSettings{
		DataShards:                 5,
		ParityShards:               5,
		Size:                       123456,
		ReadPriceRange:             PriceRange{0, 5000},
		WritePriceRange:            PriceRange{0, 5000},
		MaxChallengeCompletionTime: 1 * time.Hour,
		Duration:                   24 * 365 * time.Hour,
	}
	var mockAllBlobbers = &StorageNodes{}
	var conf = &scConfig{
		MinAllocSize:               1027,
		MinAllocDuration:           5 * time.Minute,
		MaxChallengeCompletionTime: 1 * time.Hour,
		MaxAnnualFreeAllocation:    mockMaxAnnualFreeAllocation,
		FreeAllocationSettings:     mockFreeAllocationSettings,
	}
	var now = common.Timestamp(10000)
	var mockChallengeCompletionTime = conf.MaxChallengeCompletionTime
	for i := 0; i < mockNumBlobbers; i++ {
		mockBlobber := &StorageNode{
			ID:       strconv.Itoa(i),
			Capacity: 536870912,
			Used:     73,
			Terms: Terms{
				MaxOfferDuration:        mockFreeAllocationSettings.Duration * 2,
				ReadPrice:               mockFreeAllocationSettings.ReadPriceRange.Max,
				ChallengeCompletionTime: mockChallengeCompletionTime,
			},
			LastHealthCheck: now - blobberHealthTime + 1,
		}
		mockAllBlobbers.Nodes.add(mockBlobber)
	}

	type args struct {
		ssc      *StorageSmartContract
		txn      *transaction.Transaction
		input    []byte
		balances cstate.StateContextI
	}
	type want struct {
		err    bool
		errMsg string
	}
	type parameters struct {
		assigner freeStorageAssigner
		marker   updateFreeStorageMarker
		exists   bool
	}

	setExpectations := func(t *testing.T, name string, p parameters, want want) args {
		var balances = &mocks.StateContextI{}
		var txn = &transaction.Transaction{
			ClientID:     p.marker.Recipient,
			PublicKey:    mockUserPublicKey,
			CreationDate: now,
			Value:        zcnToInt64(p.marker.FreeTokens),
		}
		txn.Hash = mockTransactionHash
		var ssc = &StorageSmartContract{
			SmartContract: sci.NewSC(ADDRESS),
		}

		p.marker.Signature, p.assigner.PublicKey = signFreeAllocationMarker(t, freeStorageMarker{
			Giver:      p.marker.Giver,
			Recipient:  p.marker.Recipient,
			FreeTokens: p.marker.FreeTokens,
			Timestamp:  p.marker.Timestamp,
		})
		input, err := json.Marshal(p.marker)
		require.NoError(t, err)
		balances.On(
			"GetTrieNode",
			freeStorageAssignerKey(ssc.ID, p.marker.Giver),
		).Return(&p.assigner, nil).Once()

		balances.On("GetTrieNode", scConfigKey(ssc.ID)).Return(conf, nil).Once()

		balances.On("GetTrieNode", ALL_BLOBBERS_KEY).Return(
			mockAllBlobbers, nil,
		).Once()

		ca := ClientAllocation{
			ClientID:    p.marker.Recipient,
			Allocations: &Allocations{},
		}
		ca.Allocations.List.add(p.marker.AllocationId)
		balances.On("GetTrieNode", ca.GetKey(ssc.ID)).Return(
			&ca, nil,
		).Once()

		var sa = StorageAllocation{
			ID:           p.marker.AllocationId,
			Owner:        p.marker.Recipient,
			Expiration:   now + 1,
			DataShards:   conf.FreeAllocationSettings.DataShards,
			ParityShards: conf.FreeAllocationSettings.ParityShards,
		}
		for _, blobber := range mockAllBlobbers.Nodes {
			balances.On(
				"GetTrieNode", blobber.GetKey(ssc.ID),
			).Return(blobber, nil).Once()
			var sp = newStakePool()
			sp.Offers[p.marker.AllocationId] = &offerPool{}
			balances.On(
				"GetTrieNode", stakePoolKey(ssc.ID, blobber.ID),
			).Return(sp, nil).Once()
			balances.On(
				"InsertTrieNode", blobber.GetKey(ssc.ID), mock.Anything,
			).Return("", nil).Once()
			balances.On(
				"InsertTrieNode", stakePoolKey(ssc.ID, blobber.ID), mock.Anything,
			).Return("", nil).Once()
			sa.BlobberDetails = append(sa.BlobberDetails, &BlobberAllocation{
				BlobberID:    blobber.ID,
				AllocationID: p.marker.AllocationId,
			})
		}
		balances.On("GetTrieNode", sa.GetKey(ssc.ID)).Return(
			&sa, nil,
		).Once()
		balances.On(
			"InsertTrieNode", sa.GetKey(ssc.ID), mock.Anything,
		).Return("", nil).Once()

		balances.On(
			"InsertTrieNode", ALL_BLOBBERS_KEY, mock.Anything,
		).Return("", nil).Once()
		balances.On(
			"GetTrieNode", writePoolKey(ssc.ID, p.marker.Recipient),
		).Return(&writePool{}, nil).Once()

		balances.On(
			"GetTrieNode", challengePoolKey(ssc.ID, p.marker.AllocationId),
		).Return(&challengePool{}, nil).Once()

		balances.On(
			"InsertTrieNode",
			freeStorageAssignerKey(ssc.ID, p.marker.Giver),
			&freeStorageAssigner{
				ClientId:    p.assigner.ClientId,
				PublicKey:   p.assigner.PublicKey,
				AnnualLimit: p.assigner.AnnualLimit,
				FreeStoragesRedeemed: append(
					p.assigner.FreeStoragesRedeemed,
					freeStorageRedeemed{
						Amount:    zcnToBalance(p.marker.FreeTokens),
						When:      txn.CreationDate,
						Timestamp: p.marker.Timestamp,
					}),
			},
		).Return("", nil).Once()

		balances.On("AddMint", &state.Mint{
			Minter:     ADDRESS,
			ToClientID: ADDRESS,
			Amount:     zcnToBalance(p.marker.FreeTokens),
		}).Return(nil).Once()

		balances.On(
			"InsertTrieNode",
			writePoolKey(ssc.ID, p.marker.Recipient),
			//mock.Anything,
			mock.MatchedBy(func(wp *writePool) bool {
				pool, found := wp.Pools.get(p.marker.AllocationId)
				require.True(t, found)
				return pool.Balance == zcnToBalance(p.marker.FreeTokens) &&
					pool.ID == mockTransactionHash &&
					pool.AllocationID == p.marker.AllocationId &&
					len(pool.Blobbers) == mockNumBlobbers
			}),
		).Return("", nil).Once()

		return args{ssc, txn, input, balances}
	}

	testCases := []struct {
		name       string
		parameters parameters
		want       want
	}{
		{
			name: "ok_no_previous",
			parameters: parameters{
				marker: updateFreeStorageMarker{
					AllocationId: mockAllocationId,
					Giver:        mockCooperationId + "ok",
					Recipient:    mockRecipient,
					FreeTokens:   mockFreeTokens,
					Timestamp:    mockTimestamp,
				},
				assigner: freeStorageAssigner{
					ClientId:    mockCooperationId + "ok",
					AnnualLimit: mockAnnualTokenLimit,
				},
			},
		},
	}
	for _, test := range testCases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			args := setExpectations(t, test.name, test.parameters, test.want)

			resp, err := args.ssc.updateFreeStorageRequest(args.txn, args.input, args.balances)

			require.True(t, mock.AssertExpectationsForObjects(t, args.balances))
			require.EqualValues(t, test.want.err, err != nil)
			if err != nil {
				require.EqualValues(t, test.want.errMsg, err.Error())
				return
			}
			require.True(t, mock.AssertExpectationsForObjects(t, args.balances))
			resp = resp
		})
	}
}

func signFreeAllocationMarker(t *testing.T, frm freeStorageMarker) (string, string) {
	var request = struct {
		Recipient  string           `json:"recipient"`
		FreeTokens float64          `json:"free_tokens"`
		Timestamp  common.Timestamp `json:"timestamp"`
	}{
		frm.Recipient, frm.FreeTokens, frm.Timestamp,
	}
	responseBytes, err := json.Marshal(&request)
	require.NoError(t, err)
	signatureScheme := chain.GetServerChain().GetSignatureScheme()
	signatureScheme.GenerateKeys()
	signature, err := signatureScheme.Sign(hex.EncodeToString(responseBytes))
	require.NoError(t, err)
	return signature, signatureScheme.GetPublicKey()
}
