package mongodb

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Builder 是一个用于构建和执行 MongoDB 操作的链式工具。
type Builder[T any] struct {
	ctx        context.Context
	collection *mongo.Collection
	filter     interface{}
	sort       interface{}
	limit      *int64
	skip       *int64
}

// WithContext 为当前链式操作设置 context.Context。
func (b *Builder[T]) WithContext(ctx context.Context) *Builder[T] {
	b.ctx = ctx
	return b
}

// Model 指定了用于解码结果的泛型类型。
// 这允许在链的后面部分获得类型提示。
// 实际上它不执行任何操作，只是为了类型转换。
func (b *Builder[T]) Model(model T) *Builder[T] {
	return &Builder[T]{
		ctx:        b.ctx,
		collection: b.collection,
		filter:     b.filter,
		sort:       b.sort,
		limit:      b.limit,
		skip:       b.skip,
	}
}

// Filter 设置查询过滤器。
func (b *Builder[T]) Filter(filter interface{}) *Builder[T] {
	b.filter = filter
	return b
}

// Sort 设置排序规则。
func (b *Builder[T]) Sort(sort interface{}) *Builder[T] {
	b.sort = sort
	return b
}

// Limit 设置查询结果的数量限制。
func (b *Builder[T]) Limit(limit int64) *Builder[T] {
	b.limit = &limit
	return b
}

// Skip 设置查询结果的跳过数量。
func (b *Builder[T]) Skip(skip int64) *Builder[T] {
	b.skip = &skip
	return b
}

// FindOne 执行查询并解码单个结果到 `result`。
// `result` 必须是一个指向结构体的指针。
func (b *Builder[T]) FindOne(result interface{}) error {
	opts := options.FindOne()
	if b.sort != nil {
		opts.SetSort(b.sort)
	}

	err := b.collection.FindOne(b.ctx, b.filter, opts).Decode(result)
	if err == mongo.ErrNoDocuments {
		return nil // 没有找到文档不作为错误返回
	}
	return err
}

// FindByID 根据 ObjectID 查找单个文档。
// `result` 必须是一个指向结构体的指针。
func (b *Builder[T]) FindByID(id string, result interface{}) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	b.filter = bson.D{{"_id", objID}}
	return b.FindOne(result)
}

// FindMany 执行查询并解码多个结果到 `results`。
// `results` 必须是一个指向切片的指针，例如 `*[]MyStruct`。
func (b *Builder[T]) FindMany(results interface{}) error {
	opts := options.Find()
	if b.sort != nil {
		opts.SetSort(b.sort)
	}
	if b.limit != nil {
		opts.SetLimit(*b.limit)
	}
	if b.skip != nil {
		opts.SetSkip(*b.skip)
	}

	cursor, err := b.collection.Find(b.ctx, b.filter, opts)
	if err != nil {
		return err
	}
	defer cursor.Close(b.ctx)

	return cursor.All(b.ctx, results)
}

// InsertOne 插入单个文档。
func (b *Builder[T]) InsertOne(document T) (*mongo.InsertOneResult, error) {
	return b.collection.InsertOne(b.ctx, document)
}

// InsertMany 批量插入多个文档。
func (b *Builder[T]) InsertMany(documents []T) (*mongo.InsertManyResult, error) {
	docs := make([]interface{}, len(documents))
	for i, d := range documents {
		docs[i] = d
	}
	return b.collection.InsertMany(b.ctx, docs)
}

// UpdateOne 更新单个文档。
func (b *Builder[T]) UpdateOne(filter, update interface{}) (*mongo.UpdateResult, error) {
	return b.collection.UpdateOne(b.ctx, filter, update)
}

// UpdateByID 根据 ObjectID 更新单个文档。
func (b *Builder[T]) UpdateByID(id string, update interface{}) (*mongo.UpdateResult, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	return b.UpdateOne(bson.D{{"_id", objID}}, update)
}

// Upsert 如果文档存在则更新，不存在则插入。
func (b *Builder[T]) Upsert(filter, update interface{}) (*mongo.UpdateResult, error) {
	opts := options.Update().SetUpsert(true)
	return b.collection.UpdateOne(b.ctx, filter, update, opts)
}

// DeleteOne 删除单个文档。
func (b *Builder[T]) DeleteOne(filter interface{}) (*mongo.DeleteResult, error) {
	return b.collection.DeleteOne(b.ctx, filter)
}

// DeleteByID 根据 ObjectID 删除单个文档。
func (b *Builder[T]) DeleteByID(id string) (*mongo.DeleteResult, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	return b.DeleteOne(bson.D{{"_id", objID}})
}

// Count 计算符合过滤条件的文档数量。
func (b *Builder[T]) Count() (int64, error) {
	return b.collection.CountDocuments(b.ctx, b.filter)
}

// Aggregate 执行聚合管道查询。
func (b *Builder[T]) Aggregate(pipeline interface{}, results interface{}) error {
	cursor, err := b.collection.Aggregate(b.ctx, pipeline)
	if err != nil {
		return err
	}
	defer cursor.Close(b.ctx)

	return cursor.All(b.ctx, results)
}
