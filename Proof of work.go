package main

import(
	"bytes" 
	"crypto/sha256"
	"strconv"
	"time"
	"fmt"
	"log"
	"math"
	"encoding/binary"
	"math/big"
)

// Block由区块头何交易两部分构成

type Block struct{
	TimeStamp int64
	PrevBlockHash []byte
	Hash []byte
	Data []byte
	Nonce int
}

//SetHash 设置当前hash块

func (b *Block) SetHash(){
	timestamp :=[]byte(strconv.FormatInt(b.TimeStamp,10))
	headers :=bytes.Join([][]byte{b.PrevBlockHash,b.Data,timestamp},[]byte{})
	hash :=sha256.Sum256(headers)
	
	b.Hash=hash[:]
}
//实现Pow算法的核心
//工作量证明的核心就是寻找有效哈希

func (pow *ProofOfWork) Run() (int,[]byte){
	var hashInt big.Int
	var hash [32]byte
	nonce :=0
	maxNonce :=math.MaxInt64
	fmt.Printf("Mining the block containing \"%s\"\n",pow.block.Data)
	
	for nonce<maxNonce{
		data :=pow.prepareData(nonce)
		
		hash=sha256.Sum256(data)
		hashInt.SetBytes(hash[:])
		
		if hashInt.Cmp(pow.target)==-1{
			fmt.Printf("\r%x",hash)
			break
		}else{
			nonce++
		}
	}
	fmt.Print("\n\n")
	return nonce,hash[:]
}

//NewBlock 用于生成新快，当前块的哈希会基于data和PrevBlockHash计算得到

func NewBlock(data string,prevBlockHash []byte) *Block{
	var hash []byte
	nonce :=0
	block :=&Block{
		TimeStamp :time.Now().Unix(),
		PrevBlockHash: prevBlockHash,
		Hash : []byte{},
		Data: []byte(data),
		Nonce: 0}
	pow :=NewProofOfWork(block)
	nonce,hash=pow.Run()
	
	block.Hash=hash[:]
	block.Nonce=nonce
	return block
}

//BlcokChain 是一个Blcok指针数组

type BlcokChain struct{
	blocks []*Block
}

//AddBlcok向链中加入一个新块,data实际上为交易

func (bc *BlcokChain) AddBlock(data string){
	prevBlock :=bc.blocks[len(bc.blocks)-1]
	newBlock :=NewBlock(data,prevBlock.Hash)
	bc.blocks =append(bc.blocks,newBlock)
}

//初始状态下，我们的链为空。NewGenesisBlock 生成创世块

func NewGenesisBlock() *Block{
	return NewBlock("Genesis Block",[]byte{})
}

//NewBlockChain 创建一个具有创世块的链

func NewBlockChain() *BlcokChain{
	return &BlcokChain{[]*Block{NewGenesisBlock()}}
}


const targetBits =10  //表示前24位为0

//每个块的工作量要得到证明，需要指向Block的指针
//target是目标，最终要找的哈希小于目标

type ProofOfWork struct{
	block *Block
	target *big.Int
}

func NewProofOfWork(b* Block) *ProofOfWork{
	target :=big.NewInt(1)
	target.Lsh(target,uint(256-targetBits))
	
	pow :=&ProofOfWork{b,target}
	
	return pow
}

func IntToHex(num int64) []byte{
	buff :=new(bytes.Buffer)
	err :=binary.Write(buff,binary.BigEndian,num)
	if err != nil{
		log.Panic(err)
	}
	return buff.Bytes()
}

//需要有数据进行哈希，准备数据
//工作量证明用到的数据有PrevBlockHash,Data,TimeStamp,targetBits,nonce

func (pow *ProofOfWork) prepareData(nonce int) []byte{
	data :=bytes.Join(
	[][]byte{
		pow.block.PrevBlockHash,
		pow.block.Data,
		IntToHex(pow.block.TimeStamp),
		IntToHex(int64(targetBits)),
		IntToHex(int64(nonce)),
		},
		[]byte{},
	)
	return data
}



//对工作量证明进行验证，只要哈希小于目标就是有效工作量
func (pow *ProofOfWork) Validata() bool{
	var hashInt big.Int
	data :=pow.prepareData(pow.block.Nonce)
	hash :=sha256.Sum256(data)
	hashInt.SetBytes(hash[:])
	
	isValid :=hashInt.Cmp(pow.target)==-1
	
	return isValid
	
}


//测试区块链是否正常工作
func main() {
	bc :=NewBlockChain()
	
	bc.AddBlock("Send 1 BTC to Ivan")
	bc.AddBlock("Send 2 more BTC to Ivan")
	
	for _,block :=range bc.blocks{
		fmt.Printf("Prev hash:%x\n",block.PrevBlockHash)
		fmt.Printf("Data :%s\n",block.Data)
		fmt.Printf("Hash :%x\n",block.Hash)
		pow :=NewProofOfWork(block)
		fmt.Printf("pow %s\n",strconv.FormatBool(pow.Validata()))
		fmt.Println()
	}
	
}










