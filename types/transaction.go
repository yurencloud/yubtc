package types

import (
	"github.com/yurencloud/yubtc/utils"
	"github.com/ethereum/go-ethereum/console"
	"bytes"
)

const (
	COINBASE_AMOUNT = 50
)

// 未使用的交易输出，即你的余额，就是你获得的余额的对应的交易。
type UnspentTxOut struct {
	TxOutId    string
	TxOutIndex int64
	Address    string
	Amount     int64
}

//一个交易事务的输入
type TxIn struct {
	TxOutId    []byte // 交易事务输出的hash
	TxOutIndex int64  // 交易事务输出的索引
	Signature  []byte // 自身签名
}

//一个交易事务的输出
type TxOut struct {
	Amount  int64    //值
	Address []byte //解锁脚本key,PubKeyHash
}

//交易事物
type Transaction struct {
	Id     []byte  //交易hash
	TxIns  []TxIn  //事物输入
	TxOuts []TxOut //事物输出
}

func (tx *Transaction) GetTransactionId() []byte {
	var txInsBytes []byte
	var txOutBytes []byte
	for _, txIn := range tx.TxIns {
		txInsBytes = utils.BytesCombine(txInsBytes, txIn.TxOutId, utils.Int64ToBytes(txIn.TxOutIndex))
	}
	for _, txOut := range tx.TxOuts {
		txOutBytes = utils.BytesCombine(txOutBytes, txOut.Address, utils.Int64ToBytes(txOut.Amount))
	}

	return utils.Sha256(utils.BytesCombine(txInsBytes, txOutBytes))
}

/*func (tx *Transaction) validateTransaction(aUnspentTxOuts []UnspentTxOut) bool {

	if bytes.Equal(tx.GetTransactionId(),tx.Id) {
		return false;
	}

const hasValidTxIns: boolean = transaction.txIns
.map((txIn) => validateTxIn(txIn, transaction, aUnspentTxOuts))
.reduce((a, b) => a && b, true);

if (!hasValidTxIns) {
console.log('some of the txIns are invalid in tx: ' + transaction.id);
return false;
}

const totalTxInValues: number = transaction.txIns
.map((txIn) => getTxInAmount(txIn, aUnspentTxOuts))
.reduce((a, b) => (a + b), 0);

const totalTxOutValues: number = transaction.txOuts
.map((txOut) => txOut.amount)
.reduce((a, b) => (a + b), 0);

if (totalTxOutValues !== totalTxInValues) {
console.log('totalTxOutValues !== totalTxInValues in tx: ' + transaction.id);
return false;
}

return true;
}

func  ValidateTxIn( txIn TxIn, transaction Transaction, aUnspentTxOuts []UnspentTxOut) bool {
const referencedUTxOut: UnspentTxOut =
aUnspentTxOuts.find((uTxO) => uTxO.txOutId === txIn.txOutId && uTxO.txOutIndex === txIn.txOutIndex);
if (referencedUTxOut == null) {
console.log('referenced txOut not found: ' + JSON.stringify(txIn));
return false;
}
const address = referencedUTxOut.address;

const key = ec.keyFromPublic(address, 'hex');
const validSignature: boolean = key.verify(transaction.id, txIn.signature);
if (!validSignature) {
console.log('invalid txIn signature: %s txId: %s address: %s', txIn.signature, transaction.id, referencedUTxOut.address);
return false;
}
return true;
};
*/