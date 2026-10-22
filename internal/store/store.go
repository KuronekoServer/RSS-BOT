// Package store は MongoDB への永続化を行います。
// コレクション名・フィールド名は旧Node.js版(mongoose)と互換です。
package store

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Target は通知先のチャンネルです。
type Target struct {
	ServerID  string `bson:"serverid"`
	ChannelID string `bson:"channelid"`
}

// Store は MongoDB のラッパーです。
type Store struct {
	client     *mongo.Client
	operations *mongo.Collection // 通知先 (旧: operation モデル)
	seen       *mongo.Collection // 通知済みリンク (旧: tmp モデル)
	feeds      *mongo.Collection // 一度でも取得したことのあるフィード
}

// New は MongoDB に接続します。
func New(ctx context.Context, uri, database string) (*Store, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("MongoDBに接続できません: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("MongoDBに接続できません: %w", err)
	}
	db := client.Database(database)
	return &Store{
		client:     client,
		operations: db.Collection("operations"),
		seen:       db.Collection("tmps"),
		feeds:      db.Collection("feeds"),
	}, nil
}

// Close は接続を閉じます。
func (s *Store) Close(ctx context.Context) error {
	return s.client.Disconnect(ctx)
}

// Register はサーバーの通知先を登録します。既に登録済みなら false を返します。
func (s *Store) Register(ctx context.Context, serverID, channelID string) (bool, error) {
	res, err := s.operations.UpdateOne(ctx,
		bson.M{"serverid": serverID},
		bson.M{"$setOnInsert": bson.M{"serverid": serverID, "channelid": channelID}},
		options.UpdateOne().SetUpsert(true),
	)
	if err != nil {
		return false, err
	}
	return res.UpsertedCount > 0, nil
}

// Unregister はサーバーの通知先を削除します。未登録なら false を返します。
func (s *Store) Unregister(ctx context.Context, serverID string) (bool, error) {
	res, err := s.operations.DeleteMany(ctx, bson.M{"serverid": serverID})
	if err != nil {
		return false, err
	}
	return res.DeletedCount > 0, nil
}

// Targets は全ての通知先を返します。
func (s *Store) Targets(ctx context.Context) ([]Target, error) {
	cur, err := s.operations.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	var out []Target
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// MarkSeen はリンクを通知済みにします。新たに記録した場合のみ true を返します。
func (s *Store) MarkSeen(ctx context.Context, link string) (bool, error) {
	res, err := s.seen.UpdateOne(ctx,
		bson.M{"link": link},
		bson.M{"$setOnInsert": bson.M{"link": link}},
		options.UpdateOne().SetUpsert(true),
	)
	if err != nil {
		return false, err
	}
	return res.UpsertedCount > 0, nil
}

// FeedKnown はフィードを過去に取得したことがあるか返します。
func (s *Store) FeedKnown(ctx context.Context, url string) (bool, error) {
	err := s.feeds.FindOne(ctx, bson.M{"url": url}).Err()
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	return err == nil, err
}

// MarkFeedKnown はフィードを取得済みとして記録します。
func (s *Store) MarkFeedKnown(ctx context.Context, url string) error {
	_, err := s.feeds.UpdateOne(ctx,
		bson.M{"url": url},
		bson.M{"$setOnInsert": bson.M{"url": url}},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

