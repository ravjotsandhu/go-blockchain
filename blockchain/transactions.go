package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
)

//Transaction struct -ID,Inputs,Outputs
type Transaction struct {
	ID      []byte
	Inputs  []TxInput
	Outputs []TxOutput
}

func CoinbaseTx(to, data string) *Transaction {
	if data == "" {
		data = fmt.Sprintf("Coins to %s", to)
	}
	txin := TxInput{[]byte{}, -1, data}
	txout := TxOutput{100, to} //100 is reward to the address for mining the block

	tx := Transaction{nil, []TxInput{txin}, []TxOutput{txout}}
	tx.SetID()
	return &tx
}

//creates a hash ID for transaction
func (tx *Transaction) SetID() {
	var encoded bytes.Buffer
	var hash [32]byte

	encode := gob.NewEncoder(&encoded)
	err := encode.Encode(tx)
	Handle(err)
	hash = sha256.Sum256(encoded.Bytes())
	tx.ID = hash[:]
}

//allows us to determine wether the transaction is coinbase or not
func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Inputs) == 1 && len(tx.Inputs[0].ID) == 0 && tx.Inputs[0].Out == -1
}

func NewTransaction(from, to string, amt int, chain *Blockchain) *Transaction {
	var inputs []TxInput
	var outputs []TxOutput

	accumualted, validOutputs := chain.FindSpendableOutputs(from, amt)

	if accumualted < amt {
		log.Panic("Error: not enough funds")
	}

	for txid, outs := range validOutputs {
		txID, err := hex.DecodeString(txid)
		Handle(err)
		for _, out := range outs {
			input := TxInput{txID, out, from}
			inputs = append(inputs, input)
		}
	}

	outputs = append(outputs, TxOutput{amt, to}) //first output is transaction
	if accumualted > amt {
		outputs = append(outputs, TxOutput{accumualted - amt, from})
	} //second output if there is any leftover token in sender account

	tx := Transaction{nil, inputs, outputs}
	tx.SetID()
	return &tx
}
