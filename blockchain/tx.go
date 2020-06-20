package blockchain

import (
	"bytes"
	"encoding/gob"

	"github.com/RavjotSandhu/GoBlockchain/wallet"
)

//TxOutput struct -Value,PubKey
type TxOutput struct {
	Value      int    //value in tokens which is assigned and locked inside of this output
	PubkeyHash []byte /* to lock the Value;it is derived using script but we don't have anyhthing
	like that, so the arbitrary key that would represent it is user's address*/
}

/*identify transaction outputs and then sort them by an unspent outputs with this new structure we create a new serialize and deserialize function
so that we can take the  actual structure, decode it into bytes and then re-encode iot back into the go structure
*/
type TxOutputs struct {
	Outputs []TxOutput
}

//TxInput struct -ID,Out,Sig
type TxInput struct {
	ID  []byte //references the transaction that output is inside in
	Out int    //references the index where output appears
	Sig []byte /*script which provides the data which is used in TxOutput's Pubkey
	because we donot have the script logic in place the sig is just going to be the user's account*/
	Pubkey []byte
}

func (in *TxInput) UsesKey(pubKeyHash []byte) bool {
	lockingHash := wallet.PublicKeyHash(in.Pubkey)
	return bytes.Compare(lockingHash, pubKeyHash) == 0
}

//locking the transaction output
func (out *TxOutput) Lock(address []byte) {
	pubKeyHash := wallet.Base58Decode(address)
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	out.PubkeyHash = pubKeyHash
}

//checks if the output has been locked with public key hash
func (out *TxOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Compare(out.PubkeyHash, pubKeyHash) == 0
}

//locking the transaction outputs that we create and also because when we pass in an address from trhe command line its a string, so we need we convert that to a slice of bytes
func NewTXOutput(value int, address string) *TxOutput {
	txo := &TxOutput{value, nil}
	txo.Lock([]byte(address))
	return txo
}

func (outs TxOutputs) Serialize() []byte {
	var buffer bytes.Buffer
	encode := gob.NewEncoder(&buffer)
	err := encode.Encode(outs)
	Handle(err)
	return buffer.Bytes()
}

func DeserializeOutputs(data []byte) TxOutputs {
	var outputs TxOutputs
	decode := gob.NewDecoder(bytes.NewReader(data))
	err := decode.Decode(&outputs)
	Handle(err)
	return outputs
}
