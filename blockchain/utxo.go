package blockchain

import (
	"bytes"
	"encoding/hex"
	"log"

	"github.com/dgraph-io/badger"
)

var (
	utxoPrefix   = []byte("utxo-")
	prefixLength = len(utxoPrefix)
)

//allows to access the database and then we can create the new layer inside of that database which will have UTXOs
type UTXOSet struct {
	Block_chain *Blockchain
}

//allows to go through the database and delete in bulk the prefix keys form the databsae and because of the way badger works we need to do this in a very specific way
func (u *UTXOSet) DeleteByPrefix(prefix []byte) {
	//creating a closure and binding it to the variable deleteKeys
	deleteKeys := func(keysForDelete [][]byte) error {
		if err := u.Block_chain.Database.Update(func(txn *badger.Txn) error {
			for _, key := range keysForDelete {
				if err := txn.Delete(key); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
		return nil
	}

	collectSize := 100000 //amt of keys that we can delete in one batch delete with badger db
	//what will happen is that if we have more than 100000 with set prefix that we are looking for it will go through delete the first 100000 and then go through next number of keys however many they are
	u.Block_chain.Database.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		keysForDelete := make([][]byte, 0, collectSize)
		keysCollected := 0
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			key := it.Item().KeyCopy(nil)
			keysForDelete = append(keysForDelete, key)
			keysCollected++
			if keysCollected == collectSize {
				if err := deleteKeys(keysForDelete); err != nil {
					log.Panic(err)
				}
				keysForDelete = make([][]byte, 0, collectSize)
				keysCollected = 0
			}
		}
		if keysCollected > 0 {
			if err := deleteKeys(keysForDelete); err != nil {
				log.Panic(err)
			}
		}
		return nil
	})
}

//clear outs th database with all the prefixes attaached to it and then rebuild the set inside ofthe database
func (u UTXOSet) Reindex() {
	db := u.Block_chain.Database
	u.DeleteByPrefix(utxoPrefix)
	UTXO := u.Block_chain.FindUTXO()
	err := db.Update(func(txn *badger.Txn) error {
		for txId, outs := range UTXO {
			key, err := hex.DecodeString(txId)
			if err != nil {
				return err
			}
			key = append(utxoPrefix, key...)
			err = txn.Set(key, outs.Serialize())
			Handle(err)
		}
		return nil
	})
	Handle(err)
}

//It enables us to create normal transactions that are not coin based
//but we do not have the ability to send the coins from one account to the other
//for this to work we need to ensure that we have all unspent outputs and then ensure that they havhe enough tokens inside of them

func (u UTXOSet) FindSpendableOutputs(pubKeyHash []byte, amt int) (int, map[string][]int) {
	unspentOuts := make(map[string][]int)
	accumulated := 0
	db := u.Block_chain.Database
	var v []byte
	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(utxoPrefix); it.ValidForPrefix(utxoPrefix); it.Next() {
			item := it.Item()
			k := item.Key()
			err1 := item.Value(func(val []byte) error {
				v = append([]byte{}, val...)
				return nil
			})
			Handle(err1)

			k = bytes.TrimPrefix(k, utxoPrefix)
			txID := hex.EncodeToString(k)
			outs := DeserializeOutputs(v)

			for outIdx, out := range outs.Outputs {
				if out.IsLockedWithKey(pubKeyHash) && accumulated < amt {
					accumulated += out.Value
					unspentOuts[txID] = append(unspentOuts[txID], outIdx)
				}
			}
		}
		return nil
	})
	Handle(err)
	return accumulated, unspentOuts
}

//goes through persistence layer and find the balance for a user based on their public key hash
// so it goes through and find all the outputs attached to that user, passes them back which we can use to find how many tokens are assigned that user
func (u UTXOSet) FindUTXO(pubKeyHash []byte) []TxOutput {
	var UTXOs []TxOutput
	var v []byte
	db := u.Block_chain.Database
	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(utxoPrefix); it.ValidForPrefix(utxoPrefix); it.Next() {
			item := it.Item()
			err1 := item.Value(func(val []byte) error {
				v = append([]byte{}, val...)
				return nil
			})
			Handle(err1)
			outs := DeserializeOutputs(v)

			for _, out := range outs.Outputs {
				if out.IsLockedWithKey(pubKeyHash) {
					UTXOs = append(UTXOs, out)
				}
			}
		}
		return nil
	})
	Handle(err)
	return UTXOs
}

//Counts how many transaction
func (u UTXOSet) CountTransactions() int {
	db := u.Block_chain.Database
	counter := 0
	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(utxoPrefix); it.ValidForPrefix(utxoPrefix); it.Next() {
			counter++
		}
		return nil
	})

	Handle(err)

	return counter
}

//updatex the UTXOset inside of our persistence layer
func (u *UTXOSet) Update(block *Block) {
	var v []byte
	db := u.Block_chain.Database
	err := db.Update(func(txn *badger.Txn) error {
		for _, tx := range block.Transactions {
			if tx.IsCoinbase() == false {
				for _, in := range tx.Inputs {
					updatedOuts := TxOutputs{}
					inID := append(utxoPrefix, in.ID...)
					item, err := txn.Get(inID)
					Handle(err)
					err1 := item.Value(func(val []byte) error {
						v = append([]byte{}, val...)
						return nil
					})
					Handle(err1)
					outs := DeserializeOutputs(v)
					for outIdx, out := range outs.Outputs {
						if outIdx != in.Out {
							updatedOuts.Outputs = append(updatedOuts.Outputs, out)
						}
					}
					if len(updatedOuts.Outputs) == 0 {
						if err := txn.Delete(inID); err != nil {
							log.Panic(err)
						}

					} else {
						if err := txn.Set(inID, updatedOuts.Serialize()); err != nil {
							log.Panic(err)
						}
					}
				}
			}

			newOutputs := TxOutputs{}
			for _, out := range tx.Outputs {
				newOutputs.Outputs = append(newOutputs.Outputs, out)
			}

			txID := append(utxoPrefix, tx.ID...)
			if err := txn.Set(txID, newOutputs.Serialize()); err != nil {
				log.Panic(err)
			}
		}

		return nil
	})
	Handle(err)
}
