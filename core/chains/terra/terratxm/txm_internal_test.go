package terratxm

import (
	"testing"
	"time"

	"github.com/pkg/errors"

	tmservicetypes "github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	tcmocks "github.com/smartcontractkit/chainlink-terra/pkg/terra/client/mocks"
	"github.com/smartcontractkit/chainlink/core/internal/testutils/pgtest"
	"github.com/smartcontractkit/chainlink/core/logger"
	"github.com/smartcontractkit/chainlink/core/services/keystore"
	"github.com/smartcontractkit/chainlink/core/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	tmtypes "github.com/tendermint/tendermint/proto/tendermint/types"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

func generateExecuteMsg(t *testing.T, from, to cosmostypes.AccAddress) []byte {
	msg1 := wasmtypes.NewMsgExecuteContract(from, to, []byte(`{"transmit":{"report_context":"","signatures":[""],"report":""}}`), cosmostypes.Coins{})
	d, err := msg1.Marshal()
	require.NoError(t, err)
	return d
}

func TestTxm(t *testing.T) {
	db := pgtest.NewSqlxDB(t)
	lggr := logger.TestLogger(t)
	ks := keystore.New(db, utils.FastScryptParams, lggr, pgtest.NewPGCfg(true))
	require.NoError(t, ks.Unlock("blah"))
	k1, err := ks.Terra().Create()
	require.NoError(t, err)
	sender1, err := cosmostypes.AccAddressFromBech32(k1.PublicKeyStr())
	require.NoError(t, err)
	k2, err := ks.Terra().Create()
	require.NoError(t, err)
	sender2, err := cosmostypes.AccAddressFromBech32(k2.PublicKeyStr())
	require.NoError(t, err)
	contract, err := cosmostypes.AccAddressFromBech32("terra1pp76d50yv2ldaahsdxdv8mmzqfjr2ax97gmue8")
	require.NoError(t, err)
	fallbackGasPrice := "0.01"
	gasLimitMultiplier := 1.5

	t.Run("single msg", func(t *testing.T) {
		tc := new(tcmocks.ReaderWriter)
		tc.On("Account", mock.Anything).Return(uint64(0), uint64(0), nil)
		tc.On("GasPrice", mock.Anything).Return(cosmostypes.NewDecCoinFromDec("uluna", cosmostypes.MustNewDecFromStr("0.01")))
		tc.On("SimulateUnsigned", mock.Anything, mock.Anything).Return(&txtypes.SimulateResponse{GasInfo: &cosmostypes.GasInfo{
			GasUsed: 1_000_000,
		}}, nil)
		tc.On("LatestBlock").Return(&tmservicetypes.GetLatestBlockResponse{Block: &tmtypes.Block{
			Header: tmtypes.Header{Height: 1},
		}}, nil)
		tc.On("CreateAndSign", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]byte{0x01}, nil)
		tc.On("Broadcast", mock.Anything, mock.Anything).Return(&txtypes.BroadcastTxResponse{
			TxResponse: &cosmostypes.TxResponse{TxHash: "0x123"},
		}, nil)
		tc.On("Tx", mock.Anything).Return(&txtypes.GetTxResponse{
			Tx:         &txtypes.Tx{},
			TxResponse: &cosmostypes.TxResponse{TxHash: "0x123"},
		}, nil)

		txm, _ := NewTxm(db, tc, fallbackGasPrice, gasLimitMultiplier, ks.Terra(), lggr, pgtest.NewPGCfg(true), nil, time.Second)

		// Enqueue a single msg, then send it in a batch
		id1, err := txm.Enqueue(contract.String(), generateExecuteMsg(t, sender1, contract))
		require.NoError(t, err)
		txm.sendMsgBatch()

		// Should be in completed state
		completed, err := txm.orm.SelectMsgsWithIDs([]int64{id1})
		require.NoError(t, err)
		require.Equal(t, 1, len(completed))
		assert.Equal(t, completed[0].State, Confirmed)
		tc.AssertExpectations(t)
	})

	t.Run("two msgs different accounts", func(t *testing.T) {
		tc := new(tcmocks.ReaderWriter)
		tc.On("Account", mock.Anything).Return(uint64(0), uint64(0), nil)
		tc.On("GasPrice", mock.Anything).Return(cosmostypes.NewDecCoinFromDec("uluna", cosmostypes.MustNewDecFromStr("0.01")))
		tc.On("SimulateUnsigned", mock.Anything, mock.Anything).Return(&txtypes.SimulateResponse{GasInfo: &cosmostypes.GasInfo{
			GasUsed: 1_000_000,
		}}, nil)
		tc.On("LatestBlock").Return(&tmservicetypes.GetLatestBlockResponse{Block: &tmtypes.Block{
			Header: tmtypes.Header{Height: 1},
		}}, nil)
		tc.On("CreateAndSign", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]byte{0x01}, nil)
		tc.On("Broadcast", mock.Anything, mock.Anything).Return(&txtypes.BroadcastTxResponse{
			TxResponse: &cosmostypes.TxResponse{TxHash: "0x123"},
		}, nil)
		tc.On("Tx", mock.Anything).Return(&txtypes.GetTxResponse{
			Tx:         &txtypes.Tx{},
			TxResponse: &cosmostypes.TxResponse{TxHash: "0x123"},
		}, nil)

		txm, _ := NewTxm(db, tc, fallbackGasPrice, gasLimitMultiplier, ks.Terra(), lggr, pgtest.NewPGCfg(true), nil, time.Second)

		id1, err := txm.Enqueue(contract.String(), generateExecuteMsg(t, sender1, contract))
		require.NoError(t, err)
		id2, err := txm.Enqueue(contract.String(), generateExecuteMsg(t, sender2, contract))
		require.NoError(t, err)
		txm.sendMsgBatch()

		// Should be in completed state
		completed, err := txm.orm.SelectMsgsWithIDs([]int64{id1, id2})
		require.NoError(t, err)
		require.Equal(t, 2, len(completed))
		assert.Equal(t, completed[0].State, Confirmed)
		assert.Equal(t, completed[1].State, Confirmed)
		tc.AssertExpectations(t)
	})

	t.Run("failed to confirm", func(t *testing.T) {
		tc := new(tcmocks.ReaderWriter)
		tc.On("Tx", mock.Anything).Return(&txtypes.GetTxResponse{
			Tx:         &txtypes.Tx{},
			TxResponse: &cosmostypes.TxResponse{TxHash: "0x123"},
		}, errors.New("not found")).Twice()
		txm, _ := NewTxm(db, tc, fallbackGasPrice, gasLimitMultiplier, ks.Terra(), lggr, pgtest.NewPGCfg(true), nil, time.Second)
		txm.confirmPollPeriod = 0 * time.Second
		txm.confirmMaxPolls = 2
		i, err := txm.orm.InsertMsg("blah", []byte{0x01})
		require.NoError(t, err)
		err = txm.confirmTx("0x123", []int64{i})
		require.NoError(t, err)
		m, err := txm.orm.SelectMsgsWithIDs([]int64{i})
		require.NoError(t, err)
		require.Equal(t, 1, len(m))
		assert.Equal(t, Errored, m[0].State)
		tc.AssertExpectations(t)
	})

	t.Run("confirm any unconfirmed", func(t *testing.T) {
		txHash1 := "0x1234"
		txHash2 := "0x1235"
		tc := new(tcmocks.ReaderWriter)
		tc.On("Tx", txHash1).Return(&txtypes.GetTxResponse{
			TxResponse: &cosmostypes.TxResponse{TxHash: txHash1},
		}, nil).Once()
		tc.On("Tx", txHash2).Return(&txtypes.GetTxResponse{
			TxResponse: &cosmostypes.TxResponse{TxHash: txHash2},
		}, nil).Once()
		txm, _ := NewTxm(db, tc, fallbackGasPrice, gasLimitMultiplier, ks.Terra(), lggr, pgtest.NewPGCfg(true), nil, time.Second)

		// Insert and broadcast 2 msgs with different txhashes.
		id1, err := txm.orm.InsertMsg("blah", []byte{0x01})
		require.NoError(t, err)
		id2, err := txm.orm.InsertMsg("blah", []byte{0x02})
		require.NoError(t, err)
		err = txm.orm.UpdateMsgsWithState([]int64{id1}, Broadcasted, &txHash1)
		require.NoError(t, err)
		err = txm.orm.UpdateMsgsWithState([]int64{id2}, Broadcasted, &txHash2)
		require.NoError(t, err)

		// Confirm them as in a restart while confirming scenario
		txm.confirmAnyUnconfirmed()
		require.NoError(t, err)
		confirmed, err := txm.orm.SelectMsgsWithIDs([]int64{id1, id2})
		require.NoError(t, err)
		require.Equal(t, 2, len(confirmed))
		assert.Equal(t, Confirmed, confirmed[0].State)
		assert.Equal(t, Confirmed, confirmed[1].State)
		tc.AssertExpectations(t)
	})
}
