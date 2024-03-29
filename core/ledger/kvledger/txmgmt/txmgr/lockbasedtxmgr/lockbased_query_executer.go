/*
Copyright IBM Corp. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package lockbasedtxmgr

import (
	"github.com/hyperledger/fabric/common/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/storageutil"
)

// LockBasedQueryExecutor is a query executor used in `LockBasedTxMgr`
type lockBasedQueryExecutor struct {
	helper *queryHelper
	txid   string
}

func newQueryExecutor(txmgr *LockBasedTxMgr, txid string) *lockBasedQueryExecutor {
	helper := newQueryHelper(txmgr, nil)
	logger.Debugf("constructing new query executor txid = [%s]", txid)
	return &lockBasedQueryExecutor{helper, txid}
}

// GetState implements method in interface `ledger.QueryExecutor`
func (q *lockBasedQueryExecutor) GetState(ns string, key string) (val []byte, err error) {
	val, _, err = q.helper.getState(ns, key)
	return
}

// GetStateMetadata implements method in interface `ledger.QueryExecutor`
func (q *lockBasedQueryExecutor) GetStateMetadata(namespace, key string) (map[string][]byte, error) {
	_, metadataBytes, err := q.helper.getState(namespace, key)
	if err != nil {
		return nil, err
	}
	metadata, err := storageutil.DeserializeMetadata(metadataBytes)
	if err != nil {
		return nil, err
	}
	return metadata, err
}

// GetStateMultipleKeys implements method in interface `ledger.QueryExecutor`
func (q *lockBasedQueryExecutor) GetStateMultipleKeys(namespace string, keys []string) ([][]byte, error) {
	return q.helper.getStateMultipleKeys(namespace, keys)
}

// GetStateRangeScanIterator implements method in interface `ledger.QueryExecutor`
// startKey is included in the results and endKey is excluded. An empty startKey refers to the first available key
// and an empty endKey refers to the last available key. For scanning all the keys, both the startKey and the endKey
// can be supplied as empty strings. However, a full scan shuold be used judiciously for performance reasons.
func (q *lockBasedQueryExecutor) GetStateRangeScanIterator(namespace string, startKey string, endKey string) (ledger.ResultsIterator, error) {
	return q.helper.getStateRangeScanIterator(namespace, startKey, endKey)
}

// ExecuteQuery implements method in interface `ledger.QueryExecutor`
func (q *lockBasedQueryExecutor) ExecuteQuery(namespace, query string) (ledger.ResultsIterator, error) {
	return q.helper.executeQuery(namespace, query)
}

// GetPrivateData implements method in interface `ledger.QueryExecutor`
func (q *lockBasedQueryExecutor) GetPrivateData(namespace, collection, key string) ([]byte, error) {
	return q.helper.getPrivateData(namespace, collection, key)
}

// GetPrivateDataMetadata implements method in interface `ledger.QueryExecutor`
func (q *lockBasedQueryExecutor) GetPrivateDataMetadata(namespace, collection, key string) (map[string][]byte, error) {
	_, metadataBytes, err := q.helper.getPrivateDataValueHash(namespace, collection, key)
	if err != nil {
		return nil, err
	}
	metadata, err := storageutil.DeserializeMetadata(metadataBytes)
	if err != nil {
		return nil, err
	}
	return metadata, err
}

// GetPrivateDataMultipleKeys implements method in interface `ledger.QueryExecutor`
func (q *lockBasedQueryExecutor) GetPrivateDataMultipleKeys(namespace, collection string, keys []string) ([][]byte, error) {
	return q.helper.getPrivateDataMultipleKeys(namespace, collection, keys)
}

// GetPrivateDataRangeScanIterator implements method in interface `ledger.QueryExecutor`
func (q *lockBasedQueryExecutor) GetPrivateDataRangeScanIterator(namespace, collection, startKey, endKey string) (ledger.ResultsIterator, error) {
	return q.helper.getPrivateDataRangeScanIterator(namespace, collection, startKey, endKey)
}

// ExecuteQueryOnPrivateData implements method in interface `ledger.QueryExecutor`
func (q *lockBasedQueryExecutor) ExecuteQueryOnPrivateData(namespace, collection, query string) (ledger.ResultsIterator, error) {
	return q.helper.executeQueryOnPrivateData(namespace, collection, query)
}

// Done implements method in interface `ledger.QueryExecutor`
func (q *lockBasedQueryExecutor) Done() {
	logger.Debugf("Done with transaction simulation / query execution [%s]", q.txid)
	q.helper.done()
}
