package main

import (
	"log"
	"testing"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/stretchr/testify/assert"
)

func TestProcessTx(t *testing.T) {
	// fetch tx from rpc
	// 创建新的RPC客户端
	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		log.Fatalf("Failed to create new RPC client: %v", err)
	}
	defer client.Shutdown()

	txid := "4385fcf8b14497d0659adccfe06ae7e38e0b5dc95ff8a13d7c62035994a0cd79"
	//getHash
	hash, err := chainhash.NewHashFromStr(txid)
	assert.Nil(t, err)
	tx, err := client.GetRawTransaction(hash)
	assert.Nil(t, err)
	//decode transaction
	processTransaction(0, tx.MsgTx(), nil)

	txid = "3abf3173d801aaf69b680d7a5dfa4ab2eef13892eeb1747d9ff861fdd051c164"
	//getHash
	hash, err = chainhash.NewHashFromStr(txid)
	assert.Nil(t, err)
	tx, err = client.GetRawTransaction(hash)
	assert.Nil(t, err)
	//decode transaction
	processTransaction(0, tx.MsgTx(), nil)

	txid = "ed776ee0701bd17e70c5bd1f7c684cd4d975cfdce4e88439153efde0636c0837"
	//getHash
	hash, err = chainhash.NewHashFromStr(txid)
	assert.Nil(t, err)
	tx, err = client.GetRawTransaction(hash)
	assert.Nil(t, err)
	//decode transaction
	processTransaction(0, tx.MsgTx(), nil)

	txid = "6385c0836bec3fdcd55589f8b0be7fd24e21cdd49a0f4c024fdc47fd3d031614"
	//getHash
	hash, err = chainhash.NewHashFromStr(txid)
	assert.Nil(t, err)
	tx, err = client.GetRawTransaction(hash)
	assert.Nil(t, err)
	//decode transaction
	processTransaction(0, tx.MsgTx(), nil)
}
