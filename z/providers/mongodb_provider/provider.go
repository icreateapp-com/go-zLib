package mongodb_provider

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/icreateapp-com/go-zLib/z/providers/config_provider"
	"github.com/icreateapp-com/go-zLib/z/providers/logger_provider"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.uber.org/fx"
)

// MongoDB 提供 MongoDB Client/Database 以及便捷的 Collection/Builder 获取能力。
type MongoDB struct {
	client *mongo.Client
	db     *mongo.Database
	log    *logger_provider.Logger
}

// MongoIn 表示 MongoDB 的 fx 入参。
type MongoIn struct {
	fx.In
	LC  fx.Lifecycle
	Cfg *config_provider.Config
	Log *logger_provider.Logger
}

// NewMongoProvider 创建 MongoDB 实例（fx Provider）。
func NewMongoProvider(in MongoIn) (*MongoDB, error) {
	host := strings.TrimSpace(in.Cfg.GetString("mongodb.host", in.Cfg.GetString("mongodb_provider.host", "")))
	port := strings.TrimSpace(in.Cfg.GetString("mongodb.port", in.Cfg.GetString("mongodb_provider.port", "")))
	dbName := strings.TrimSpace(in.Cfg.GetString("mongodb.dbname", in.Cfg.GetString("mongodb_provider.dbname", "")))
	username := strings.TrimSpace(in.Cfg.GetString("mongodb.username", in.Cfg.GetString("mongodb_provider.username", "")))
	password := strings.TrimSpace(in.Cfg.GetString("mongodb.password", in.Cfg.GetString("mongodb_provider.password", "")))
	authSource := strings.TrimSpace(in.Cfg.GetString("mongodb.auth_source", in.Cfg.GetString("mongodb_provider.auth_source", "")))
	connectTimeout := in.Cfg.GetDuration("mongodb.connect_timeout", in.Cfg.GetDuration("mongodb_provider.connect_timeout", 10*time.Second))
	ping := in.Cfg.GetBool("mongodb.ping", in.Cfg.GetBool("mongodb_provider.ping", true))

	if host == "" || port == "" || dbName == "" {
		return nil, fmt.Errorf("mongodb_provider.host/mongodb_provider.port/mongodb_provider.dbname are required")
	}

	var uri string
	if username != "" && password != "" {
		username = url.QueryEscape(username)
		password = url.QueryEscape(password)
		if authSource == "" {
			authSource = "admin"
		}
		uri = fmt.Sprintf("mongodb://%s:%s@%s:%s/%s?authSource=%s", username, password, host, port, dbName, authSource)
	} else {
		uri = fmt.Sprintf("mongodb://%s:%s/%s", host, port, dbName)
	}

	p := &MongoDB{log: in.Log}

	in.LC.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			cctx, cancel := context.WithTimeout(ctx, connectTimeout)
			defer cancel()

			client, err := mongo.Connect(cctx, options.Client().ApplyURI(uri))
			if err != nil {
				return err
			}
			if ping {
				if err := client.Ping(cctx, readpref.Primary()); err != nil {
					_ = client.Disconnect(context.Background())
					return err
				}
			}

			p.client = client
			p.db = client.Database(dbName)
			if p.log != nil {
				p.log.Infow("provider[mongodb_provider] enabled", "db", dbName, "host", host, "port", port)
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if p.client == nil {
				return nil
			}
			if p.log != nil {
				p.log.Infow("provider[mongodb_provider] stopped")
			}
			return p.client.Disconnect(ctx)
		},
	})

	return p, nil
}

// DB 返回 *mongo.Database。
func (p *MongoDB) DB() *mongo.Database { return p.db }

// Client 返回 *mongo.Client。
func (p *MongoDB) Client() *mongo.Client { return p.client }

// GetCollection 返回指定名称的 *mongo.Collection。
func (p *MongoDB) GetCollection(name string) *mongo.Collection {
	if p == nil || p.db == nil {
		return nil
	}
	return p.db.Collection(name)
}

// Collection 是开始一个链式调用的入口点。
func (p *MongoDB) Collection(name string) *Builder[any] {
	return &Builder[any]{
		collection: p.GetCollection(name),
	}
}

// MongoProviderModule 提供 MongoDB 的 fx 模块。
var MongoProviderModule = fx.Options(
	fx.Provide(NewMongoProvider),
)
