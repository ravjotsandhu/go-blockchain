package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/RavjotSandhu/GoBlockchain/wallet"
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
	txin := TxInput{[]byte{}, -1, nil, []byte(data)}
	txout := NewTXOutput(100, to) //100 is reward to the address for mining the block

	tx := Transaction{nil, []TxInput{txin}, []TxOutput{*txout}}
	tx.ID = tx.Hash()
	return &tx
}

//takes the transactions and create a hash which we use as a transaction ID
func (tx *Transaction) Hash() []byte {
	var hash [32]byte
	txCopy := *tx
	txCopy.ID = []byte{}
	hash = sha256.Sum256(txCopy.Serialize())
	return hash[:]
}

//takes our transaction and serialize into bytes
func (tx Transaction) Serialize() []byte {
	var encoded bytes.Buffer
	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	if err != nil {
		log.Panic(err)
	}
	return encoded.Bytes()
}

//allows us to determine wether the transaction is coinbase or not
func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Inputs) == 1 && len(tx.Inputs[0].ID) == 0 && tx.Inputs[0].Out == -1
}

func (tx *Transaction) Sign(privKey ecdsa.PrivateKey, prevTXs map[string]Transaction) {
	if tx.IsCoinbase() {
		return
	}
	for _, in := range tx.Inputs {
		if prevTXs[hex.EncodeToString(in.ID)].ID == nil {
			log.Panic("ERROR: Previous transaction is not correct")
		}
	}
	txCopy := tx.TrimmedCopy()

	for inId, in := range txCopy.Inputs {
		prevTX := prevTXs[hex.EncodeToString(in.ID)]
		txCopy.Inputs[inId].Sig = nil
		txCopy.Inputs[inId].Pubkey = prevTX.Outputs[in.Out].PubkeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Inputs[inId].Pubkey = nil

		r, s, err := ecdsa.Sign(rand.Reader, &privKey, txCopy.ID)
		Handle(err)
		signature := append(r.Bytes(), s.Bytes()...)
		tx.Inputs[inId].Sig = signature
	}
}

func (tx *Transaction) TrimmedCopy() Transaction {
	var inputs []TxInput
	var outputs []TxOutput
	for _, in := range tx.Inputs {
		inputs = append(inputs, TxInput{in.ID, in.Out, nil, nil})
	}
	for _, out := range tx.Outputs {
		outputs = append(outputs, TxOutput{out.Value, out.PubkeyHash})
	}
	txCopy := Transaction{tx.ID, inputs, outputs}
	return txCopy
}

func (tx *Transaction) Verify(prevTXs map[string]Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}
	for _, in := range tx.Inputs {
		if prevTXs[hex.EncodeToString(in.ID)].ID == nil {
			log.Panic("Previous transaction not correct")
		}
	}

	txCopy := tx.TrimmedCopy()
	curve := elliptic.P256()
	for inId, in := range tx.Inputs {
		prevTx := prevTXs[hex.EncodeToString(in.ID)]
		txCopy.Inputs[inId].Sig = nil
		txCopy.Inputs[inId].Pubkey = prevTx.Outputs[in.Out].PubkeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Inputs[inId].Pubkey = nil

		r := big.Int{}
		s := big.Int{}
		sigLen := len(in.Sig)
		r.SetBytes(in.Sig[:(sigLen / 2)])
		s.SetBytes(in.Sig[(sigLen / 2):])
		x := big.Int{}
		y := big.Int{}
		keyLen := len(in.Pubkey)
		x.SetBytes(in.Pubkey[:(keyLen / 2)])
		y.SetBytes(in.Pubkey[(keyLen / 2):])
		rawPubKey := ecdsa.PublicKey{curve, &x, &y}
		if ecdsa.Verify(&rawPubKey, txCopy.ID, &r, &s) == false {
			return false
		}
	}
	return true
}

func (tx Transaction) String() string {
	var lines []string
	lines = append(lines, fmt.Sprintf("--- Transaction %x:", tx.ID))
	for i, input := range tx.Inputs {
		lines = append(lines, fmt.Sprintf("     Input %d:", i))
		lines = append(lines, fmt.Sprintf("       TXID:     %x", input.ID))
		lines = append(lines, fmt.Sprintf("       Out:       %d", input.Out))
		lines = append(lines, fmt.Sprintf("       Sig: %x", input.Sig))
		lines = append(lines, fmt.Sprintf("       Pubkey:    %x", input.Pubkey))
	}
	for i, output := range tx.Outputs {
		lines = append(lines, fmt.Sprintf("     Output %d:", i))
		lines = append(lines, fmt.Sprintf("       Value:  %d", output.Value))
		lines = append(lines, fmt.Sprintf("       Script: %x", output.PubkeyHash))
	}
	return strings.Join(lines, "\n")
}

func NewTransaction(from, to string, amt int, UTXO *UTXOSet) *Transaction {
	var inputs []TxInput
	var outputs []TxOutput

	wallets, err := wallet.CreateWallets()
	Handle(err)
	w := wallets.GetWallet(from)
	pubKeyHash := wallet.PublicKeyHash(w.PublicKey)
	accumualted, validOutputs := UTXO.FindSpendableOutputs(pubKeyHash, amt)
	if accumualted < amt {
		log.Panic("Error: not enough funds")
	}

	for txid, outs := range validOutputs {
		txID, err := hex.DecodeString(txid)
		Handle(err)
		for _, out := range outs {
			input := TxInput{txID, out, nil, w.PublicKey}
			inputs = append(inputs, input)
		}
	}

	outputs = append(outputs, *NewTXOutput(amt, to)) //first output is transaction
	if accumualted > amt {
		outputs = append(outputs, *NewTXOutput(accumualted-amt, from))
	} //second output if there is any leftover token in sender account

	tx := Transaction{nil, inputs, outputs}
	tx.ID = tx.Hash()
	UTXO.Block_chain.SignTransaction(&tx, w.PrivateKey)
	return &tx
}
