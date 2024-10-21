package main

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	mongoClient *mongo.Client
	database    *mongo.Database
	collection  *mongo.Collection
	apiItems    []bson.M // 在内存中缓存数据
	mu          sync.Mutex
)

// 初始化MongoDB连接
func initMongo() {
	var err error
	mongoURI := "mongodb://username:passwd@IP:port/DatabaseName?authSource=admin" // 请替换为你的MongoDB连接信息

	mongoClient, err = mongo.Connect(context.TODO(), options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal(err)
	}

	database = mongoClient.Database("DatabaseName")    // 请替换为你的数据库名称
	collection = database.Collection("CollectionName") // 请替换为你的集合名称

	// 预先加载数据
	preloadData()
}

// 预先加载数据到内存
func preloadData() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := collection.Find(ctx, bson.D{{}})
	if err != nil {
		log.Fatal(err)
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &apiItems); err != nil {
		log.Fatal(err)
	}
}

// 处理API请求
func randomApiHandler(w http.ResponseWriter, r *http.Request) {
	if len(apiItems) == 0 {
		http.Error(w, "No data found", http.StatusNotFound)
		return
	}

	rand.Seed(time.Now().UnixNano())
	randomIndex := rand.Intn(len(apiItems))
	responseData := apiItems[randomIndex]

	// 获取访问信息
	accessTime := time.Now().Format(time.RFC3339)
	accessIP := r.RemoteAddr

	// 打印访问日志到终端
	log.Printf("访问日志: IP: %s, Time: %s, 返回内容: %v\n", accessIP, accessTime, responseData)

	// 返回JSON响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseData)
}

// 处理主页请求
func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	// 读取 index.html 文件
	file, err := os.ReadFile("index.html")
	if err != nil {
		http.Error(w, "无法读取首页", http.StatusInternalServerError)
		return
	}

	// 返回 index.html 文件内容
	w.Write(file)
}

// 主函数
func main() {
	initMongo()
	defer mongoClient.Disconnect(context.TODO())

	http.HandleFunc("/", indexHandler) // 处理主页请求
	http.HandleFunc("/api", randomApiHandler)

	// 启动8080端口的API服务
	go func() {
		log.Println("API服务器在8080端口启动...")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatalf("ListenAndServe: %v", err)
		}
	}()

	select {} // 阻塞主线程
}
