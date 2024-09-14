package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

// Структура транзакции
type Transaction struct {
	Type              string  // Тип транзакции (например, "CONTENT_CREATION", "TOKEN_TRANSFER")
	Sender            string  // Отправитель
	Receiver          string  // Получатель (если применимо)
	Amount            float64 // Сумма (если это перевод токенов)
	ContentID         string  // Идентификатор контента (если применимо)
	Metadata          string  // Описание или содержание
	Timestamp         string  // Время транзакции
	ProofOfContribution float64 // Вклад в систему PoC
}

// Структура блока
type Block struct {
	Index        int           // Порядковый номер блока
	Timestamp    string        // Время создания блока
	Transactions []Transaction // Список транзакций в блоке
	Hash         string        // Хеш блока
	PrevHash     string        // Хеш предыдущего блока
}

// Блокчейн (список блоков)
var Blockchain []Block
var mutex = &sync.Mutex{} // Для синхронизации доступа к блокчейну

// Создание хеша для блока
func calculateHash(block Block) string {
	record := fmt.Sprintf("%d%s%v%s", block.Index, block.Timestamp, block.Transactions, block.PrevHash)
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

// Создание нового блока
func createBlock(prevBlock Block, transactions []Transaction) Block {
	block := Block{
		Index:        prevBlock.Index + 1,
		Timestamp:    time.Now().String(),
		Transactions: transactions,
		PrevHash:     prevBlock.Hash,
	}
	block.Hash = calculateHash(block)
	return block
}

// Генерация транзакции
func createTransaction(txType, sender, receiver, contentID, metadata string, amount, proofOfContribution float64) Transaction {
	return Transaction{
		Type:              txType,
		Sender:            sender,
		Receiver:          receiver,
		Amount:            amount,
		ContentID:         contentID,
		Metadata:          metadata,
		Timestamp:         time.Now().String(),
		ProofOfContribution: proofOfContribution,
	}
}

// Инициализация блокчейна с генезис-блоком
func initBlockchain() {
	genesisBlock := Block{
		Index:        0,
		Timestamp:    time.Now().String(),
		Transactions: nil,
		Hash:         "",
		PrevHash:     "",
	}
	genesisBlock.Hash = calculateHash(genesisBlock)
	Blockchain = append(Blockchain, genesisBlock)
}

// Хранение блокчейна в файле
func saveBlockchain() {
	file, _ := json.MarshalIndent(Blockchain, "", " ")
	_ = ioutil.WriteFile("blockchain.json", file, 0644)
}

// Загрузка блокчейна из файла
func loadBlockchain() {
	file, err := ioutil.ReadFile("blockchain.json")
	if err != nil {
		log.Println("Ошибка загрузки блокчейна, создаем новый генезис-блок.")
		initBlockchain()
		return
	}
	json.Unmarshal(file, &Blockchain)
}

// P2P-сеть: передача данных о блокчейне между узлами
func handleConnection(conn net.Conn) {
	defer conn.Close()
	buf, _ := ioutil.ReadAll(conn)
	receivedBlockchain := []Block{}
	_ = json.Unmarshal(buf, &receivedBlockchain)

	// Блокировка и синхронизация блокчейна
	mutex.Lock()
	if len(receivedBlockchain) > len(Blockchain) {
		Blockchain = receivedBlockchain
		saveBlockchain()
	}
	mutex.Unlock()
}

// Запуск P2P узла
func startServer(port string) {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	for {
		conn, _ := ln.Accept()
		go handleConnection(conn)
	}
}

// Подключение к другому узлу в сети
func connectToNode(nodeAddress string) {
	conn, err := net.Dial("tcp", nodeAddress)
	if err != nil {
		log.Println("Ошибка подключения к узлу", nodeAddress)
		return
	}
	defer conn.Close()

	// Отправка текущего блокчейна
	mutex.Lock()
	data, _ := json.Marshal(Blockchain)
	mutex.Unlock()
	conn.Write(data)
}

// Создание нового блока с транзакциями и отправка другим узлам
func createNewBlock(transactions []Transaction) {
	mutex.Lock()
	prevBlock := Blockchain[len(Blockchain)-1]
	newBlock := createBlock(prevBlock, transactions)
	Blockchain = append(Blockchain, newBlock)
	saveBlockchain()
	mutex.Unlock()

	// Распространение нового блока по сети
	for _, nodeAddress := range []string{"localhost:5001", "localhost:5002"} {
		go connectToNode(nodeAddress)
	}
}

func main() {
	// Загрузка блокчейна из файла
	loadBlockchain()

	// Пример транзакции
	tx1 := createTransaction("CONTENT_CREATION", "Alice", "", "post1", "My first post on Amalgam", 0, 10)
	tx2 := createTransaction("TOKEN_TRANSFER", "Bob", "Charlie", "", "", 100, 0)

	// Создание нового блока
	createNewBlock([]Transaction{tx1, tx2})

	// Запуск узла P2P на порту 5000
	go startServer("5000")

	// Подключение к другим узлам (например, узлы на портах 5001, 5002)
	for _, nodeAddress := range []string{"localhost:5001", "localhost:5002"} {
		go connectToNode(nodeAddress)
	}

	// Запуск программы (ожидание завершения)
	select {}
}
