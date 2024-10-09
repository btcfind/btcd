package main

import (
	"fmt"
	"log"
	"math/big"
	"os"

	btcec "github.com/btcsuite/btcd/btcec1"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/cmd/scanblock/data"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/jessevdk/go-flags"
	"github.com/syndtr/goleveldb/leveldb"
	"google.golang.org/protobuf/proto"
)

var Conf *Config

type signData struct {
	txHash *chainhash.Hash
	vin    int
	rindex int
	height int
	r, s   *big.Int
}

func saveSignData(db *leveldb.DB, data *data.SignData) error {
	// 序列化 SignData
	serializedData, err := proto.Marshal(data)
	if err != nil {
		return err
	}
	key := []byte(fmt.Sprintf("data:%d:%d:%d:%x", data.Height, data.Vin, data.Rindex, data.R))
	// 使用 R 作为键
	return db.Put(key, serializedData, nil)
}

func signDataProto(d *signData) *data.SignData {
	return &data.SignData{
		Hash:   d.txHash.CloneBytes(),
		Vin:    int32(d.vin),
		Rindex: int32(d.rindex),
		Height: int32(d.height),
		R:      d.r.Bytes(),
		S:      d.s.Bytes(),
	}
}

func protoToSignData(d *data.SignData) *signData {
	txHash, err := chainhash.NewHash(d.Hash)
	if err != nil {
		log.Fatalf("Failed to create hash from bytes: %v", err)
	}
	return &signData{
		txHash: txHash,
		vin:    int(d.Vin),
		rindex: int(d.Rindex),
		height: int(d.Height),
		r:      new(big.Int).SetBytes(d.R),
		s:      new(big.Int).SetBytes(d.S),
	}
}

func getConfig() *rpcclient.ConnConfig {
	// 解析命令行选项
	var opts Options
	_, err := flags.Parse(&opts)
	//如果无法解析，使用默认的配置
	if err != nil {
		opts = Options{
			ConfigFile: "scan.conf",
		}
	}

	// 从配置文件中加载配置
	Conf, err = loadConfig(opts.ConfigFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 读取证书文件
	certs, err := os.ReadFile(Conf.RPCCert)
	if err != nil {
		log.Fatalf("Failed to read cert file: %v", err)
	}

	// 创建新的RPC客户端配置
	connCfg := &rpcclient.ConnConfig{
		Host:         Conf.RPCListen,
		User:         Conf.RPCUser,
		Pass:         Conf.RPCPass,
		HTTPPostMode: true,  // 使用HTTP POST模式
		DisableTLS:   false, // 启用TLS
		Certificates: certs,
	}
	return connCfg
}

var connCfg = getConfig()
var maxHeight = int64(0)

func main() {
	// 创建新的RPC客户端
	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		log.Fatalf("Failed to create new RPC client: %v", err)
	}
	defer client.Shutdown()

	// 打开 LevelDB 数据库
	db, err := leveldb.OpenFile(Conf.DBPath, nil)
	if err != nil {
		log.Fatalf("Failed to open ldb: %v", err)
	}
	// 获取区块链的最新高度
	blockCount, err := client.GetBlockCount()
	if err != nil {
		log.Fatalf("Failed to get block count: %v", err)
	}

	// 遍历区块链中的所有区块
	ch := make(chan signData)
	go processRS(ch, db)
	//遍历数据库中某个前缀的数据
	iter := db.NewIterator(nil, nil)
	iter.Seek([]byte("data:"))
	for iter.Next() {
		data := &data.SignData{}
		err := proto.Unmarshal(iter.Value(), data)
		if err != nil {
			log.Fatalf("Failed to unmarshal data: %v", err)
		}
		if int64(data.Height) > maxHeight {
			maxHeight = int64(data.Height)
		}
		ch <- *protoToSignData(data)
	}
	if err := iter.Error(); err != nil {
		log.Fatalf("Failed to iterate: %v", err)
	}
	iter.Release()

	//获取当前区块高度
	for height := maxHeight; height <= blockCount; height++ {
		blockHash, err := client.GetBlockHash(height)
		if err != nil {
			log.Fatalf("Failed to get block hash at height %d: %v", height, err)
		}

		block, err := client.GetBlock(blockHash)
		if err != nil {
			log.Fatalf("Failed to get block at height %d: %v", height, err)
		}
		if height%1000 == 0 {
			log.Println("block height: ", height)
		}
		// 遍历区块中的所有交易
		for _, tx := range block.Transactions {
			processTransaction(int(height), tx, ch)
		}
	}
	close(ch)
}

func processRS(ch chan signData, db *leveldb.DB) {
	var sigdata = make(map[*big.Int]int)
	for data := range ch {
		sigdata[data.r]++
		if sigdata[data.r] > 1 {
			fmt.Printf("R: %x\n", data.r.Bytes())
			fmt.Printf("S: %x\n", data.s.Bytes())
		}
		if data.height < int(maxHeight) {
			continue
		}
		//save key-value to leveldb
		err := saveSignData(db, signDataProto(&data))
		if err != nil {
			log.Fatalf("Failed to save data: %v", err)
		}
	}
}

func processTransaction(height int, tx *wire.MsgTx, ch chan signData) {
	txHash := tx.TxHash()
	// 遍历交易中的所有输入
	for index, txIn := range tx.TxIn {
		// 获取签名脚本
		sigScript := txIn.SignatureScript
		// 解析见证数据
		witness := txIn.Witness
		var r, s []*big.Int
		if len(witness) > 0 {
			// 处理包含见证数据的交易
			r, s = processWitnessData(witness)
		} else {
			// 处理不包含见证数据的交易
			r, s = processNonWitnessData(sigScript)
		}
		for i := 0; i < len(r); i++ {
			if ch != nil {
				ch <- signData{
					txHash: &txHash,
					vin:    index,
					rindex: i,
					height: height,
					r:      r[i],
					s:      s[i],
				}
			} else {
				fmt.Printf("R: %x\n", r[i].Bytes())
				fmt.Printf("S: %x\n", s[i].Bytes())
			}
		}
	}
}

func processWitnessData(witness wire.TxWitness) (r, s []*big.Int) {
	// 假设见证数据的最后两个元素是签名和公钥
	sig := witness[len(witness)-2]
	// 解析签名
	signature, err := btcec.ParseDERSignature(sig, btcec.S256())
	if err != nil {
		log.Printf("Failed to parse DER signature: %v", err)
		return
	}
	return []*big.Int{signature.R}, []*big.Int{signature.S}
}

func processNonWitnessData(sigScript []byte) (r, s []*big.Int) {
	// 使用 MakeScriptTokenizer 解析签名脚本
	tokenizer := txscript.MakeScriptTokenizer(0, sigScript)
	for tokenizer.Next() {
		data := tokenizer.Data()
		if len(data) == 71 {
			// 解析签名
			signature, err := btcec.ParseDERSignature(data, btcec.S256())
			if err != nil {
				log.Printf("Failed to parse DER signature: %v", err)
				continue
			}
			r = append(r, signature.R)
			s = append(s, signature.S)
		}
	}

	if err := tokenizer.Err(); err != nil {
		log.Printf("Failed to tokenize script: %v", err)
	}
	return r, s
}
