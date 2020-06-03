package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"log"
	"math"
	"math/big"
)

const Difficulty = 12

//we take data from block
type ProofofWork struct {
	Block  *Block   // our block
	Target *big.Int // The First few bytes must contain 0s
}

func NewProof(b *Block) *ProofofWork {
	target := big.NewInt(1)                  //we create our target by casting number 1
	target.Lsh(target, uint(256-Difficulty)) //256 is number of bytes in our hash

	pow := &ProofofWork{b, target}

	return pow
}

//we create a cohesive set of bytes which we return from this function
func (pow *ProofofWork) InitData(nonce int) []byte {
	data := bytes.Join(
		[][]byte{
			pow.Block.PrevHash,
			pow.Block.HashTransactions(),
			ToByte(int64(nonce)),
			ToByte(int64(Difficulty)),
		},
		[]byte{},
	)

	return data
}

func ToByte(num int64) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num) //A ByteOrder specifies how to convert byte sequences into 16-, 32-, or 64-bit unsigned integers.
	if err != nil {
		log.Panic(err)
	}
	return buff.Bytes()
}

func (pow *ProofofWork) Run() (int, []byte) {
	var intHash big.Int
	var hash [32]byte

	nonce := 0

	for nonce < math.MaxInt64 {
		data := pow.InitData(nonce)
		hash = sha256.Sum256(data)

		fmt.Printf("\r%x", hash)
		intHash.SetBytes(hash[:])

		if intHash.Cmp(pow.Target) == -1 {
			break //because hash is actually less than target we are looking for which means we have actually signed the block
		} else {
			nonce++ //otherwise we increase nonce and repeat process again until if statement is true
		}

	}
	fmt.Println()

	return nonce, hash[:]
}

/*
after we have run proof of work allowing us to derive
the hash which met the target we wanted then we will
be able to run this cycle one more time to show that
hash is valid
*/
func (pow *ProofofWork) Validate() bool {
	var intHash big.Int

	data := pow.InitData(pow.Block.Nonce)

	hash := sha256.Sum256(data)
	intHash.SetBytes(hash[:])

	return intHash.Cmp(pow.Target) == -1
}
