package polybft

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/0xPolygon/polygon-edge/blockchain"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/bitmap"
	bls "github.com/0xPolygon/polygon-edge/consensus/polybft/signer"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/wallet"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/umbracle/ethgo"
)

func TestHelpers_isEpochEndingBlock_DeltaNotEmpty(t *testing.T) {
	t.Parallel()

	validators := newTestValidators(3).getPublicIdentities()
	bitmap := bitmap.Bitmap{}
	bitmap.Set(0)

	delta := &ValidatorSetDelta{
		Added:   validators[1:],
		Removed: bitmap,
	}

	extra := &Extra{Validators: delta, Checkpoint: &CheckpointData{EpochNumber: 2}}
	blockNumber := uint64(20)

	isEndOfEpoch, err := isEpochEndingBlock(blockNumber, extra, new(blockchainMock))
	require.NoError(t, err)
	require.True(t, isEndOfEpoch)
}

func TestHelpers_isEpochEndingBlock_NoBlock(t *testing.T) {
	t.Parallel()

	blockchainMock := new(blockchainMock)
	blockchainMock.On("GetHeaderByNumber", mock.Anything).Return(&types.Header{}, false)

	extra := &Extra{Checkpoint: &CheckpointData{EpochNumber: 2}, Validators: &ValidatorSetDelta{}}
	blockNumber := uint64(20)

	isEndOfEpoch, err := isEpochEndingBlock(blockNumber, extra, blockchainMock)
	require.ErrorIs(t, blockchain.ErrNoBlock, err)
	require.False(t, isEndOfEpoch)
}

func TestHelpers_isEpochEndingBlock_EpochsNotTheSame(t *testing.T) {
	t.Parallel()

	blockchainMock := new(blockchainMock)

	nextBlockExtra := &Extra{Checkpoint: &CheckpointData{EpochNumber: 3}, Validators: &ValidatorSetDelta{}}
	nextBlock := &types.Header{
		Number:    21,
		ExtraData: append(make([]byte, ExtraVanity), nextBlockExtra.MarshalRLPTo(nil)...),
	}

	blockchainMock.On("GetHeaderByNumber", mock.Anything).Return(nextBlock, true)

	extra := &Extra{Checkpoint: &CheckpointData{EpochNumber: 2}, Validators: &ValidatorSetDelta{}}
	blockNumber := uint64(20)

	isEndOfEpoch, err := isEpochEndingBlock(blockNumber, extra, blockchainMock)
	require.NoError(t, err)
	require.True(t, isEndOfEpoch)
}

func TestHelpers_isEpochEndingBlock_EpochsAreTheSame(t *testing.T) {
	t.Parallel()

	blockchainMock := new(blockchainMock)

	nextBlockExtra := &Extra{Checkpoint: &CheckpointData{EpochNumber: 2}, Validators: &ValidatorSetDelta{}}
	nextBlock := &types.Header{
		Number:    16,
		ExtraData: append(make([]byte, ExtraVanity), nextBlockExtra.MarshalRLPTo(nil)...),
	}

	blockchainMock.On("GetHeaderByNumber", mock.Anything).Return(nextBlock, true)

	extra := &Extra{Checkpoint: &CheckpointData{EpochNumber: 2}, Validators: &ValidatorSetDelta{}}
	blockNumber := uint64(15)

	isEndOfEpoch, err := isEpochEndingBlock(blockNumber, extra, blockchainMock)
	require.NoError(t, err)
	require.False(t, isEndOfEpoch)
}

func createTestKey(t *testing.T) *wallet.Key {
	t.Helper()

	return wallet.NewKey(wallet.GenerateAccount())
}

func createSignature(t *testing.T, accounts []*wallet.Account, hash types.Hash) *Signature {
	t.Helper()

	var signatures bls.Signatures

	var bmp bitmap.Bitmap
	for i, x := range accounts {
		bmp.Set(uint64(i))

		src, err := x.Bls.Sign(hash[:])
		require.NoError(t, err)

		signatures = append(signatures, src)
	}

	aggs, err := signatures.Aggregate().Marshal()
	require.NoError(t, err)

	return &Signature{AggregatedSignature: aggs, Bitmap: bmp}
}

func generateStateSyncEvents(t *testing.T, eventsCount int, startIdx uint64) []*StateSyncEvent {
	t.Helper()

	stateSyncEvents := make([]*StateSyncEvent, eventsCount)
	for i := 0; i < eventsCount; i++ {
		stateSyncEvents[i] = &StateSyncEvent{
			ID:     startIdx + uint64(i),
			Sender: ethgo.Address(types.StringToAddress(fmt.Sprintf("0x5%d", i))),
			Data:   generateRandomBytes(t),
		}
	}

	return stateSyncEvents
}

// generateRandomBytes generates byte array with random data of 32 bytes length
func generateRandomBytes(t *testing.T) (result []byte) {
	t.Helper()

	result = make([]byte, types.HashLength)
	_, err := rand.Reader.Read(result)
	require.NoError(t, err, "Cannot generate random byte array content.")

	return
}

// getEpochNumber returns epoch number for given blockNumber and epochSize.
// Epoch number is derived as a result of division of block number and epoch size.
// Since epoch number is 1-based (0 block represents special case zero epoch),
// we are incrementing result by one for non epoch-ending blocks.
func getEpochNumber(t *testing.T, blockNumber, epochSize uint64) uint64 {
	t.Helper()

	if isEndOfPeriod(blockNumber, epochSize) {
		return blockNumber / epochSize
	}

	return blockNumber/epochSize + 1
}
