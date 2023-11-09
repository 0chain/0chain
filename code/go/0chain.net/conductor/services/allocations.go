package services

import (
	"0chain.net/conductor/stores"
	"0chain.net/conductor/types"
	"0chain.net/conductor/utils"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

var allocationStore = stores.GetAllocationStore()

type AllocationService struct {
	allocation *types.Allocation
	baseUrl    string
}

func NewAllocationService(baseUrl string) *AllocationService {
	return &AllocationService{
		allocation: &types.Allocation{},
		baseUrl:    baseUrl,
	}
}

func (s *AllocationService) StoreAllocationsData() error {
	allocationID, err := getAllocationIDFromFile()
	if err != nil {
		return err
	}

	remoteAllocation, err := s.getRemoteAllocation(allocationID)
	if err != nil {
		return err
	}

	allocationStore.Add(*remoteAllocation)

	return nil
}

func (s *AllocationService) CompareRollBackTokens() (bool, error) {
	allocationID, err := getAllocationIDFromFile()
	if err != nil {
		return false, err
	}

	remoteAllocation, err := s.getRemoteAllocation(allocationID)
	if err != nil {
		return false, err
	}

	localAllocation, err := allocationStore.GetLatest()
	if err != nil {
		return false, err
	}

	if remoteAllocation.MovedToChallenge == localAllocation.MovedToChallenge {
		return false, errors.New("active blobbers did not commit write markers")
	}

	if remoteAllocation.MovedBack == localAllocation.MovedBack {
		return false, errors.New("active blobbers did not submit rollback WM")
	}

	if remoteAllocation.MovedToChallenge-remoteAllocation.MovedBack != 0 {
		return false, errors.New("incorrect rollback tokens distributed")
	}

	return true, nil
}

func (s *AllocationService) getRemoteAllocation(allocationID string) (*types.Allocation, error) {
	url := fmt.Sprintf("%v/allocation?allocation=%s", s.baseUrl, allocationID)

	resp, err := utils.HttpGet(url, map[string]string{})
	if err != nil {
		log.Println("Error getting allocation from remote", err)
		return nil, err
	}

	alloc := &types.Allocation{}
	err = json.Unmarshal(resp, alloc)
	if err != nil {
		return nil, err
	}

	return alloc, nil
}

func getAllocationIDFromFile() (string, error) {
	// Read allocationID from file
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	filePath := filepath.Join(homeDir, ".zcn", "allocation.txt")
	allocationIDBytes, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	allocationID := string(allocationIDBytes)

	return allocationID, nil
}
