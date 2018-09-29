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
	"github.com/boltdb/bolt"
	"encoding/gob"
	"flag"
	"os"
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

//tip 存储最后一个快的哈希
//db存储数据库连接

type BlcokChain struct{
	tip []byte
	db *bolt.DB
}

//加入区块时，需要将区块持久化到数据库中

func (bc *BlcokChain) AddBlock(data string){
	var lastHash []byte
	//这是BoltDB的另一个类型
	//获取最后一个块的哈希生成新块的哈希
	err :=bc.db.View(func(tx *bolt.Tx) error{
		b :=tx.Bucket([]byte(blocksBucket))
		lastHash = b.Get([]byte("l"))
		return nil
	})
	
	newBlock :=NewBlock(data,lastHash)
	
	err = bc.db.Update(func(tx *bolt.Tx) error{
		b :=tx.Bucket([]byte(blocksBucket))
		err :=b.Put(newBlock.Hash,newBlock.Serialize())
		err =b.Put([]byte("l"),newBlock.Hash)
		bc.tip=newBlock.Hash
		return nil
	})
	
}
//检查区块链，使用一个区块链迭代器读取他们
type BlockchainIterator struct{
	currentHash []byte
	db *bolt.DB
}

func (bc *Blockchain) Iterator() *BlockchainIterator{
	bci :=&BlockchainIterator{bc.tip,bc.db}
	
	return bci
}

//返回链中的下一个块

func (i *BlockchainIterator) Next() *Block{
	var block *Block
	
	err :=i.db.View(func(tx *bolt.Tx) error{
		b :=tx.Bucket([]byte(blocksBucket))
		encodedBlock :=b.Get(i.currentHash)
		block =DeserializeBlock(encodedBlock)
		
		return nil
	})
	
	i.currentHash=block.prevBlockHash
	
	return block
}






const targetBits =10  //表示前24位为0

//将Block序列化为一个字节数组
func (b* Block) Serialize() []byte{
	var result bytes.Buffer
	encoder :=gob.NewEnCoder(&result)
	
	err :=encoder.Encoder(b)
	
	return result.bytes()
}

//将字节数组反序列化为Block
func DeserializeBlock(d []byte) *Block{
	var block Block
	
	decoder :=gob.NewDecoder(bytes.NewReader(d))
	err :=decoder.Decode(&block)
	return &block
}

//持久化
func NewBlockChain1() *Blockchain{
	var tip []byte
	//打开一个BoltDB文件的标准做法
	db,err :=bolt.Open(dbFile,0600,nil)
	err =db.Update(func(tx *bolt.Tx) error{
		//函数的核心，先获取存储快的bucket
		b :=tx.Bucket([]byte(blocksBucket))
		//如果数据库中不存在区块链就创建一个，否则直接读取最后一个快的哈希
		if b==nil{
			fmt.Println("No existing blockchain found. create a new one...")
			genesis :=NewGenesisBlock()
			b, err :=tx.CreateBucket([]byte(blocksBucket))
			err =b.Put(genesis.Hash,genesis.Serialize())
			err =b.Put([]byte("l"),genesis.Hash)
			tip=genesis.Hash
		}else{
			tip=b.Get([]byte("l"))
		}
		return nil
	})
	//创建BlockChain的一个新方式
	bc :=BlockChain{tip,db}	
	return &bc
}


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

//所有的命令行相关的操作会通过CLI结构进行处理

type CLI struct{
	bc *BlcokChain
}

//入口是Run函数

func (cli *CLI) Run(){
	cli.validateArgs()
	//使用标准库里边的flag采纳数来解析命令行参数
	//创建两个子命令：AddBlock 何 printchain
	addBlockCmd :=flag.NewFlagSet("addblock",flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("printchain",flag.ExitOnError)
	//给addblock添加-data命令，printchain没有任何标志
	addBlockData := addBlockCmd.String("data","","Block Data")
	//使用用户提供的命令，解析flag子命令
	switch os.Args[1]{
		case "addblock":
			err :=addBlockCmd.parse(os.Args[2:])
		case "printchain":
			err := printChainCmd.Parse(os.Args[2:])
		default:
			cli.printUsage()
			os.Exit(1)
	}
	//接着检查解析哪一个是子命令，调用相关函数
	if addBlockCmd.Parsed(){
		if *addBlockData==""{
			addBlockCmd.Usage()
			os.Exit(1)
		}
		cli.bc.AddBlock(*addBlockData)
	}
	if printChainCmd.parse(){
		cli.printchain()
	}
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
	
	defer bc.db.Close()
	
	cli :=CLI{bc}
	cli.Run()
	
}










