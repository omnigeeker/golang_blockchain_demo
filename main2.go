package main

import (
    "bufio"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "io"
    "log"
    "net"
    "os"
    "strconv"
    "time"

    "github.com/davecgh/go-spew/spew"
    "github.com/joho/godotenv"
)

type Block struct {
    Index     int			// 这个块在整个链中的位置
    Timestamp string		// 块生成的时间戳
    BPM       int			// 每分钟的心跳数
    Hash      string		// 这个块通过SHA256的散列值
    PrevHash  string		// 代表前一个块的SHA256散列值
}

var Blockchain []Block

// 用来计算给定的数据的 SHA256 散列值
func calculateHash(block Block) string {
    record := string(block.Index) + block.Timestamp + string(block.BPM) + block.PrevHash
    h := sha256.New()
    h.Write([]byte(record))
    hashed := h.Sum(nil)
    return hex.EncodeToString(hashed)
}

// 便携一个生成块
func generateBlock(oldBlock Block, BPM int) (Block, error) {
    var newBlock Block

    t := time.Now()
    newBlock.Index = oldBlock.Index + 1
    newBlock.Timestamp = t.String()
    newBlock.BPM = BPM
    newBlock.PrevHash = oldBlock.Hash
    newBlock.Hash = calculateHash(newBlock)

    return newBlock, nil
}

// 校验块
func isBlockValid(newBlock, oldBlock Block) bool {
    if oldBlock.Index+1 != newBlock.Index {
        return false
    }
    if oldBlock.Hash != newBlock.PrevHash {
        return false
    }
    if calculateHash(newBlock) != newBlock.Hash {
        return false
    }
    return true
}

// 将本地的过期的链切换成最新的链, 最长链法则
func replaceChain(newBlocks []Block) {
    if len(newBlocks) > len(Blockchain) {
        Blockchain = newBlocks
    }
}

// 声明一个全局变量 bcServer ，以 channel 的形式来接受块。
var bcServer chan []Block

func main() {
    err := godotenv.Load()
    if err != nil {
        log.Fatal(err)
    }

    bcServer = make(chan []Block)

    // create genesis block
    t := time.Now()
    genesisBlock := Block{0, t.String(), 0, "", ""}
    spew.Dump(genesisBlock)
    Blockchain = append(Blockchain, genesisBlock)


	// start TCP and serve TCP server
	server, err := net.Listen("tcp", ":"+os.Getenv("ADDR"))
	if err != nil {
		log.Fatal(err)
	}
	defer server.Close()

	for {
		conn, err := server.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go handleConn(conn)
	}

}

func handleConn(conn net.Conn) {
    io.WriteString(conn, "Enter a new BPM:")

    scanner := bufio.NewScanner(conn)

    // take in BPM from stdin and add it to blockchain after 
    // conducting necessary validation
    go func() {
        for scanner.Scan() {
            bpm, err := strconv.Atoi(scanner.Text())
            if err != nil {
                log.Printf("%v not a number: %v", scanner.Text(), err)
                continue
            }
            newBlock, err := generateBlock(
                                Blockchain[len(Blockchain)-1], bpm)
            if err != nil {
                log.Println(err)
                continue
            }
            if isBlockValid(newBlock, Blockchain[len(Blockchain)-1]) {
                newBlockchain := append(Blockchain, newBlock)
                replaceChain(newBlockchain)
            }

            bcServer <- Blockchain
            io.WriteString(conn, "\nEnter a new BPM:")
        }
    }()
    
	defer conn.Close()
	
	// simulate receiving broadcast
	go func() {
		for {
			time.Sleep(30 * time.Second)
			output, err := json.Marshal(Blockchain)
			if err != nil {
				log.Fatal(err)
			}
			io.WriteString(conn, string(output))
		}
	}()

	for _ = range bcServer {
		spew.Dump(Blockchain)
	}
}

