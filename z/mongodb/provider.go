package mongodb

import (
	"context"
	"fmt"
	"github.com/icreateapp-com/go-zLib/z"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"net/url"
	"sync"
	"time"
)

type mongodb struct {
	mgoClient *mongo.Client
	mgoDB     *mongo.Database
	once      sync.Once
}

var MongoDB mongodb

// Init 初始化 MongoDB 客户端和数据库连接。
func (m *mongodb) Init() {
	m.once.Do(func() {
		host, err := z.Config.String("config.mongodb.host")
		if err != nil {
			panic(fmt.Errorf("mongodb host not configured: %w", err))
		}
		port, err := z.Config.String("config.mongodb.port")
		if err != nil {
			panic(fmt.Errorf("mongodb port not configured: %w", err))
		}
		dbName, err := z.Config.String("config.mongodb.dbname")
		if err != nil {
			panic(fmt.Errorf("mongodb dbname not configured: %w", err))
		}
		username, _ := z.Config.String("config.mongodb.username") // username 和 password 可以为空
		password, _ := z.Config.String("config.mongodb.password")
		authSource, _ := z.Config.String("config.mongodb.auth_source") // 认证数据库，默认为 admin

		var uri string
		if username != "" && password != "" {
			// 使用 URL 编码处理特殊字符
			username = url.QueryEscape(username)
			password = url.QueryEscape(password)

			// 如果未指定认证数据库，默认使用 admin
			if authSource == "" {
				authSource = "admin"
			}
			uri = fmt.Sprintf("mongodb://%s:%s@%s:%s/%s?authSource=%s", username, password, host, port, dbName, authSource)
		} else {
			uri = fmt.Sprintf("mongodb://%s:%s/%s", host, port, dbName)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
		if err != nil {
			panic(fmt.Errorf("failed to connect to mongodb: %w", err))
		}

		// Ping a primary node to verify that the connection is successful.
		if err := client.Ping(ctx, readpref.Primary()); err != nil {
			panic(fmt.Errorf("failed to ping mongodb: %w", err))
		}

		m.mgoClient = client
		m.mgoDB = client.Database(dbName)

		z.Info.Printf("Successfully connected to MongoDB, database: %s", dbName)
	})
}

// DB 返回全局的 *mongo.Database 实例。
func (m *mongodb) DB() *mongo.Database {
	if m.mgoDB == nil {
		panic("MongoDB client is not initialized. Please call Init() in main.")
	}
	return m.mgoDB
}

// Client 返回全局的 *mongo.Client 实例。
func (m *mongodb) Client() *mongo.Client {
	if m.mgoClient == nil {
		panic("MongoDB client is not initialized. Please call Init() in main.")
	}
	return m.mgoClient
}

// GetCollection 返回指定名称的 *mongo.Collection 实例。
func (m *mongodb) GetCollection(name string) *mongo.Collection {
	if m.mgoDB == nil {
		panic("MongoDB client is not initialized. Please call Init() in main.")
	}
	return m.mgoDB.Collection(name)
}

// Disconnect 断开与 MongoDB 的连接。
func (m *mongodb) Disconnect(ctx context.Context) error {
	if m.mgoClient != nil {
		z.Info.Printf("Disconnecting from MongoDB...")
		return m.mgoClient.Disconnect(ctx)
	}
	return nil
}

// Collection 是开始一个链式调用的入口点。
func (m *mongodb) Collection(name string) *Builder[any] {
	return &Builder[any]{
		collection: m.GetCollection(name),
	}
}
