package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"

	"github.com/RavjotSandhu/GoBlockchain/blockchain"
)

type CommandLine struct {
	blockchain *blockchain.Blockchain
}

func (cli *CommandLine) printUsage() {
	fmt.Println("Usage:")
	fmt.Println(" add -block BLOCK_DATA - add a block to the chain")
	fmt.Println(" print - Prints the blocks in the chain")
}

//it will allow us to validate any argument that we pass through command line
func (cli *CommandLine) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		runtime.Goexit()
	}
} //one of the small downsides to the badger is that it needs to properly garbage collect the values and keys before it shuts down so that if our app shuts without proper closing of database it can corrupt the data

//another  method for command line 'addBlock'
func (cli *CommandLine) addBlock(data string) {
	cli.blockchain.AddBlock(data)
	fmt.Println("Added Block!")
}

//a command to print out our blockchain and it uses Iterator() to itertate through database
func (cli *CommandLine) printChain() {
	iter := cli.blockchain.Iterator()

	for {
		block := iter.Next()

		fmt.Printf("Prev. hash: %x\n", block.PrevHash)
		fmt.Printf("Data: %s\n", block.Data)
		fmt.Printf("Hash: %x\n", block.Hash)
		pow := blockchain.NewProof(block)
		fmt.Printf("PoW: %s\n", strconv.FormatBool(pow.Validate()))
		fmt.Println()

		if len(block.PrevHash) == 0 {
			break
		} //when we reach genesis block it won't have any previous hash so the length would be zero and we exit loop
	}
}

//in this run() method for our command line struct just call all other methods.This is the method which we call in the main function to add the command line utility
func (cli *CommandLine) run() {
	cli.validateArgs()
	//after program execution
	//if user types in 'add' we create a new flag set
	addBlockCmd := flag.NewFlagSet("add", flag.ExitOnError)
	//if user types in 'print' we create a new flag set
	printChainCmd := flag.NewFlagSet("print", flag.ExitOnError)
	//
	addBlockData := addBlockCmd.String("block", "", "Block data")

	//we are going to call it on the first argument of the original call to the program
	switch os.Args[1] {
	case "add":
		//we want to add a block to our blockchain
		//we call Parse on our addBlockCmd
		//we can parse all of the arguments which come after the first argument in our argument list then we can handle the error
		err := addBlockCmd.Parse(os.Args[2:])
		blockchain.Handle(err)

	case "print":
		//if user types in 'print' for the first arg of the arguments list
		//we call Parse on our addBlockCmd
		//we can parse all of the arguments which come after the first argument in our argument list then we can handle the error
		err := printChainCmd.Parse(os.Args[2:])
		blockchain.Handle(err)

	default:
		//if user types nothing or anything else
		cli.printUsage()
		runtime.Goexit()
	}
	//if the parsed flags dont give us an error they give a boolean value which we can check by calling the parsed method on flag
	if addBlockCmd.Parsed() {
		if *addBlockData == "" {
			addBlockCmd.Usage()
			runtime.Goexit()
		}
		cli.addBlock(*addBlockData)
	}
	// for printChainCmd we just want to check wether it has been parsed and if it has been parsed we can just run the printChain()
	if printChainCmd.Parsed() {
		cli.printChain()
	}
}

func main() {
	defer os.Exit(0)
	chain := blockchain.InitBlockchain()
	defer chain.Database.Close()

	cli := CommandLine{chain}
	cli.run()
}
