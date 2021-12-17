// Code generated by mockery v2.8.0. DO NOT EDIT.

package mocks

import (
	common "github.com/ethereum/go-ethereum/common"
	bulletprooftxmanager "github.com/smartcontractkit/chainlink/core/services/bulletprooftxmanager"

	gorm "gorm.io/gorm"

	mock "github.com/stretchr/testify/mock"
)

// TxManager is an autogenerated mock type for the TxManager type
type TxManager struct {
	mock.Mock
}

// CreateEthTransaction provides a mock function with given fields: db, fromAddress, toAddress, payload, gasLimit, meta, strategy
func (_m *TxManager) CreateEthTransaction(db *gorm.DB, fromAddress common.Address, toAddress common.Address, payload []byte, gasLimit uint64, meta interface{}, strategy bulletprooftxmanager.TxStrategy) (bulletprooftxmanager.EthTx, error) {
	ret := _m.Called(db, fromAddress, toAddress, payload, gasLimit, meta, strategy)

	var r0 bulletprooftxmanager.EthTx
	if rf, ok := ret.Get(0).(func(*gorm.DB, common.Address, common.Address, []byte, uint64, interface{}, bulletprooftxmanager.TxStrategy) bulletprooftxmanager.EthTx); ok {
		r0 = rf(db, fromAddress, toAddress, payload, gasLimit, meta, strategy)
	} else {
		r0 = ret.Get(0).(bulletprooftxmanager.EthTx)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*gorm.DB, common.Address, common.Address, []byte, uint64, interface{}, bulletprooftxmanager.TxStrategy) error); ok {
		r1 = rf(db, fromAddress, toAddress, payload, gasLimit, meta, strategy)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}