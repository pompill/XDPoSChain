package tests

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/stretchr/testify/assert"
)

func TestYourTurnInitialV2(t *testing.T) {
	config := params.TestXDPoSMockChainConfigWithV2EngineEpochSwitch
	blockchain, _, parentBlock, _ := PrepareXDCTestBlockChain(t, int(config.XDPoS.Epoch)-1, config)
	adaptor := blockchain.Engine().(*XDPoS.XDPoS)

	// Insert block 900
	t.Logf("Inserting block with propose at 900...")
	blockCoinbaseA := "0xaaa0000000000000000000000000000000000900"
	//Get from block validator error message
	merkleRoot := "35999dded35e8db12de7e6c1471eb9670c162eec616ecebbaf4fddd4676fb930"
	header := &types.Header{
		Root:       common.HexToHash(merkleRoot),
		Number:     big.NewInt(int64(900)),
		ParentHash: parentBlock.Hash(),
		Coinbase:   common.HexToAddress(blockCoinbaseA),
		Extra:      common.Hex2Bytes("d7830100018358444388676f312e31352e38856c696e757800000000000000000278c350152e15fa6ffc712a5a73d704ce73e2e103d9e17ae3ff2c6712e44e25b09ac5ee91f6c9ff065551f0dcac6f00cae11192d462db709be3758ccef312ee5eea8d7bad5374c6a652150515d744508b61c1a4deb4e4e7bf057e4e3824c11fd2569bcb77a52905cda63b5a58507910bed335e4c9d87ae0ecdfafd400"),
	}
	block900, err := insertBlock(blockchain, header)
	if err != nil {
		t.Fatal(err)
	}

	// YourTurn is called before mine first v2 block
	b, err := adaptor.YourTurn(blockchain, block900.Header(), common.HexToAddress("xdc0278C350152e15fa6FFC712a5A73D704Ce73E2E1"))
	assert.Nil(t, err)
	assert.False(t, b)
	b, err = adaptor.YourTurn(blockchain, block900.Header(), common.HexToAddress("xdc03d9e17Ae3fF2c6712E44e25B09Ac5ee91f6c9ff"))
	assert.Nil(t, err)
	// round=1, so masternode[1] has YourTurn = True
	assert.True(t, b)
	assert.Equal(t, adaptor.EngineV2.GetCurrentRound(), utils.Round(1))

	snap, err := adaptor.EngineV2.GetSnapshot(blockchain, block900.Header())
	assert.Nil(t, err)
	assert.NotNil(t, snap)
	masterNodes := adaptor.EngineV1.GetMasternodesFromCheckpointHeader(block900.Header())
	for i := 0; i < len(masterNodes); i++ {
		assert.Equal(t, masterNodes[i].Hex(), snap.NextEpochMasterNodes[i].Hex())
	}
}

func TestUpdateMasterNodes(t *testing.T) {
	config := params.TestXDPoSMockChainConfigWithV2EngineEpochSwitch
	blockchain, backend, currentBlock, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, int(config.XDPoS.Epoch+config.XDPoS.Gap)-1, config, 0)
	adaptor := blockchain.Engine().(*XDPoS.XDPoS)
	x := adaptor.EngineV2
	snap, err := x.GetSnapshot(blockchain, currentBlock.Header())

	assert.Nil(t, err)
	assert.Equal(t, int(snap.Number), 450)

	// Insert block 1350
	t.Logf("Inserting block with propose at 1350...")
	blockCoinbaseA := "0xaaa0000000000000000000000000000000001350"
	tx, err := voteTX(37117, 0, acc1Addr.String())
	if err != nil {
		t.Fatal(err)
	}
	//Get from block validator error message
	merkleRoot := "46234e9cd7e85a267f7f0435b15256a794a2f6d65cc98cdbd21dcd10a01d9772"
	header := &types.Header{
		Root:       common.HexToHash(merkleRoot),
		Number:     big.NewInt(int64(1350)),
		ParentHash: currentBlock.Hash(),
		Coinbase:   common.HexToAddress(blockCoinbaseA),
	}
	// insert header validator
	err = generateSignature(backend, adaptor, header)
	if err != nil {
		t.Fatal(err)
	}
	parentBlock, err := insertBlockTxs(blockchain, header, []*types.Transaction{tx})
	assert.Nil(t, err)

	t.Logf("Inserting block from 1351 to 1800...")
	for i := 1351; i <= 1800; i++ {
		blockCoinbase := fmt.Sprintf("0xaaa000000000000000000000000000000000%4d", i)
		//Get from block validator error message
		merkleRoot := "46234e9cd7e85a267f7f0435b15256a794a2f6d65cc98cdbd21dcd10a01d9772"
		header = &types.Header{
			Root:       common.HexToHash(merkleRoot),
			Number:     big.NewInt(int64(i)),
			ParentHash: parentBlock.Hash(),
			Coinbase:   common.HexToAddress(blockCoinbase),
		}
		err = generateSignature(backend, adaptor, header)
		if err != nil {
			t.Fatal(err)
		}
		block, err := insertBlock(blockchain, header)
		if err != nil {
			t.Fatal(err)
		}
		parentBlock = block
	}

	snap, err = x.GetSnapshot(blockchain, parentBlock.Header())

	assert.Nil(t, err)
	assert.False(t, snap.IsMasterNodes(acc3Addr))
	assert.True(t, snap.IsMasterNodes(acc1Addr))
	assert.Equal(t, int(snap.Number), 1350)
}

func TestPrepare(t *testing.T) {
	config := params.TestXDPoSMockChainConfigWithV2EngineEpochSwitch
	blockchain, _, currentBlock, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, int(config.XDPoS.Epoch), config, 0)
	adaptor := blockchain.Engine().(*XDPoS.XDPoS)

	adaptor.YourTurn(blockchain, currentBlock.Header(), common.HexToAddress("xdc0278C350152e15fa6FFC712a5A73D704Ce73E2E1"))

	tstamp := time.Now().Unix()
	header901 := &types.Header{
		ParentHash: currentBlock.Hash(),
		Number:     big.NewInt(int64(901)),
		GasLimit:   params.TargetGasLimit,
		Time:       big.NewInt(tstamp),
	}

	err := adaptor.Prepare(blockchain, header901)
	assert.Nil(t, err)

	snap, err := adaptor.EngineV2.GetSnapshot(blockchain, currentBlock.Header())
	if err != nil {
		t.Fatal(err)
	}

	validators := []byte{}
	for _, v := range snap.NextEpochMasterNodes {
		validators = append(validators, v[:]...)
	}
	assert.Equal(t, validators, header901.Validators)

	var decodedExtraField utils.ExtraFields_v2
	err = utils.DecodeBytesExtraFields(header901.Extra, &decodedExtraField)
	assert.Nil(t, err)
	assert.Equal(t, utils.Round(1), decodedExtraField.Round)
	assert.Equal(t, utils.Round(0), decodedExtraField.QuorumCert.ProposedBlockInfo.Round)
}
