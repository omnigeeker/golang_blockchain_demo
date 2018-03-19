package main

import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "io"
	"log"
	"os"
    "net/http"
    "time"

    "github.com/davecgh/go-spew/spew"
    "github.com/gorilla/mux"
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

/**
 * 后面是创建HTTP请求的部分了
 */

// Web服务 使用Gorilla/mux 包
func run() error {
    mux := makeMuxRouter()
    httpAddr := os.Getenv("ADDR")
    log.Println("Listening on ", os.Getenv("ADDR"))
    s := &http.Server{
        Addr:           ":" + httpAddr,
        Handler:        mux,
        ReadTimeout:    10 * time.Second,
        WriteTimeout:   10 * time.Second,
        MaxHeaderBytes: 1 << 20,
    }

    if err := s.ListenAndServe(); err != nil {
        return err
    }

    return nil
}

// 对于HTTP服务器的 Handler
func makeMuxRouter() http.Handler {
    muxRouter := mux.NewRouter()
    muxRouter.HandleFunc("/", handleGetBlockchain).Methods("GET")
    muxRouter.HandleFunc("/", handleWriteBlock).Methods("POST")
    return muxRouter
}

// Get请求的 Handler
func handleGetBlockchain(w http.ResponseWriter, r *http.Request) {
    bytes, err := json.MarshalIndent(Blockchain, "", "  ")
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    io.WriteString(w, string(bytes))
}

// POST请求的 payload
type Message struct {
    BPM int
}

// Post请求的 Handler
func handleWriteBlock(w http.ResponseWriter, r *http.Request) {
    var m Message

    decoder := json.NewDecoder(r.Body)
    if err := decoder.Decode(&m); err != nil {
        respondWithJSON(w, r, http.StatusBadRequest, r.Body)
        return
    }
    defer r.Body.Close()

    newBlock, err := generateBlock(Blockchain[len(Blockchain)-1], m.BPM)
    if err != nil {
        respondWithJSON(w, r, http.StatusInternalServerError, m)
        return
    }
    if isBlockValid(newBlock, Blockchain[len(Blockchain)-1]) {
        newBlockchain := append(Blockchain, newBlock)
        replaceChain(newBlockchain)
        spew.Dump(Blockchain)
    }

    respondWithJSON(w, r, http.StatusCreated, newBlock)
}

// POST处理完成后，返回给客户端一个响应
func respondWithJSON(w http.ResponseWriter, r *http.Request, code int, payload interface{}) {
    response, err := json.MarshalIndent(payload, "", "  ")
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        w.Write([]byte("HTTP 500: Internal Server Error"))
        return
    }
    w.WriteHeader(code)
    w.Write(response)
}

// main 函数
func main() {
    err := godotenv.Load()
    if err != nil {
        log.Fatal(err)
    }

    go func() {
        t := time.Now()
		genesisBlock := Block{0, t.String(), 0, "", ""}	 	// 创世区块
        spew.Dump(genesisBlock)
        Blockchain = append(Blockchain, genesisBlock)
    }()
    log.Fatal(run())

}

