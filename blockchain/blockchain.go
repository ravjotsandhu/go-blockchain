package blockchain

import (
	"fmt"

	"github.com/dgraph-io/badger"
)

const (
	path = "/blocks"
)

type Blockchain struct {
	LastHash []byte
	Database *badger.DB
}

type BlockchainIterator struct {
	CurrentHash []byte
	Database    *badger.DB
}

func InitBlockchain() *Blockchain {
	var lastHash []byte
	//with this option struct we want to specify where we want our database files to be stored
	opts := badger.DefaultOptions(path)
	opts.Dir = path      //stores keys and meta data
	opts.ValueDir = path //here database stores all of the values but here it does not matter because the folder is same

	db, err := badger.Open(opts) //it returns a tuple with a pointer to the database and an error
	Handle(err)

	err = db.Update(func(txn *badger.Txn) error {
		/*
			check if there is a blockchain already stored or not
			if there is alreadya blockchainthen we will create a new blockchain instance in memory and we will get the
			last hash of our blockchian in our disk database and we will push to this instance in memory
			the reason why the last hash is important is that it helps derive a new block in our blockchain
			if there is no existing blockchain in our we will create a genesis block,store it in the database,then we will save the genesis block's hash as the lastblock hash in our database
			then we will create a new blockchain instance with lasthash pointing towards the genesis block
		*/
		if _, err := txn.Get([]byte("lh")); err == badger.ErrKeyNotFound {
			fmt.Println("No existing blockchain found")
			genesis := Genesis()
			fmt.Println("Genesis proved")
			err = txn.Set(genesis.Hash, genesis.Serialize())
			Handle(err)
			err = txn.Set([]byte("lh"), genesis.Hash)
			lastHash = genesis.Hash
			return err
		} else {
			item, err := txn.Get([]byte("lh"))
			Handle(err)
			err1 := item.Value(func(val []byte) error {
				lastHash = append([]byte{}, val...)
				return nil
			})
			Handle(err1)
			return err
		}
	})
	Handle(err)
	blockchain := Blockchain{lastHash, db}
	return &blockchain
}

func (chain *Blockchain) AddBlock(data string) {
	var lastHash []byte

	err := chain.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		Handle(err)
		err1 := item.Value(func(val []byte) error {
			lastHash = append([]byte{}, val...)
			return nil
		})
		Handle(err1)
		return err
	})
	Handle(err)

	newBlock := CreateBlock(data, lastHash)

	err = chain.Database.Update(func(txn *badger.Txn) error {
		err := txn.Set(newBlock.Hash, newBlock.Serialize())
		Handle(err)
		err = txn.Set([]byte("lh"), newBlock.Hash)
		chain.LastHash = newBlock.Hash
		return err
	})
	Handle(err)
}

//converting the Blockchain struct into the BlockchainIterator struct
func (chain *Blockchain) Iterator() *BlockchainIterator {
	iter := &BlockchainIterator{chain.LastHash, chain.Database}
	return iter
}

func (iter *BlockchainIterator) Next() *Block {
	var block *Block

	err := iter.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get(iter.CurrentHash)
		Handle(err)
		var encodedBlock []byte
		err1 := item.Value(func(val []byte) error {
			encodedBlock = append([]byte{}, val...)
			return nil
		})
		Handle(err1)
		block = Deserialize(encodedBlock)
		return err
	})
	Handle(err)
	iter.CurrentHash = block.PrevHash
	return block
}
