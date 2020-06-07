package blockchain

//TxOutput struct -Value,PubKey
type TxOutput struct {
	Value  int    //value in tokens which is assigned and locked inside of this output
	Pubkey string /* a value needed to unlock the tokens inside the value field
	it is derived using script but we don't have anyhthing like that, so the arbitrary key that would represent it is user's address*/
}

//TxInput struct -ID,Out,Sig
type TxInput struct {
	ID  []byte //references the transaction that output is inside in
	Out int    //references the index where output appears
	Sig string /*script which provides the data which is used in TxOutput's Pubkey
	because we donot have the script logic in place the sig is just going to be the user's account*/
}

func (in *TxInput) CanUnlock(data string) bool {
	return in.Sig == data
}

func (out *TxOutput) CanBeUnlocked(data string) bool {
	return out.Pubkey == data
}
