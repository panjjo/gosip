package main

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	log "github.com/sirupsen/logrus"
)

// DBModel default key
type DBModel struct {
	ID      string `json:"id" bson:"_id"`
	AddTime int64  `json:"addtime" bson:"addtime"`
	DelTime int64  `json:"deltime" bson:"deltime"`
	UpTime  int64  `json:"uptime" bson:"uptime"`
}

var (
	dbClient DBClient
	database string
)

// DBConfig config
type DBConfig struct {
	Addr   string `json:"addr" yaml:"addr" mapstructure:"addr"`
	DBName string `json:"db" yaml:"db" mapstructure:"db"`
	Retry  *bool  `json:"retry" yaml:"retry"`
}

// DBClient DBClient
type DBClient interface {
	SetDB(string) DBClient
	Insert(string, interface{}) error
	Update(string, interface{}, interface{}) error
	UpdateMany(string, interface{}, interface{}) (int64, int64, error)
	UpdateResult(string, interface{}, interface{}) (int64, int64, error)
	Upsert(string, interface{}, interface{}) (bool, error)
	Get(string, interface{}, interface{}) error
	Count(string, interface{}) (int64, error)
	Find(string, interface{}, int64, int64, string, bool, interface{}) (int64, error)
	Del(string, interface{}) error
	DelMany(string, interface{}) (int64, int64, error)
}

var (
	// ErrObjNotArray ErrObjNotArray
	ErrObjNotArray = fmt.Errorf("对象不是数组")
	// ErrRecordNouFound ErrRecordNouFound
	ErrRecordNouFound = mongo.ErrNoDocuments
)

// M M
type M = bson.M

// Client client
type mgoClient struct {
	client *mongo.Client
	db     string
	ctx    context.Context
}

var mongoClient *mongo.Client

// InitDB InitDB
func InitDB(config DBConfig) {
	database = config.DBName
	ops := options.Client()
	if config.Retry != nil {
		ops.SetRetryWrites(*config.Retry)
	}
	ops.ApplyURI(config.Addr)

	client, err := mongo.NewClient(ops)
	if err != nil {
		log.Fatal(err)
	}
	err = client.Connect(context.TODO())
	if err != nil {
		log.Fatal(err)
	}
	mongoClient = client
	log.Infoln("Init Mogo")
}

// NewClient NewClient
func NewClient() DBClient {
	return &mgoClient{client: mongoClient, db: database, ctx: context.Background()}
}

// SetDB SetDB
func (c *mgoClient) SetDB(db string) DBClient {
	c.db = db
	return c
}

//SetSession SetSession
func (c *mgoClient) SetSession(session mongo.SessionContext) DBClient {
	c.ctx = session
	return c
}

// Insert Insert
func (c *mgoClient) Insert(tb string, obj interface{}) error {
	now := time.Now()
	defer func() {
		log.Debugf("[MgoInsert][%v][%s]\n", time.Since(now), tb)
	}()
	cl := c.client.Database(c.db).Collection(tb)
	_, err := cl.InsertOne(c.ctx, obj)
	return err
}

// Update 更新
func (c *mgoClient) Update(tb string, query, update interface{}) error {
	_, _, err := c.UpdateResult(tb, query, update)
	return err
}

// UpdateResult 更新一条数据并返回 找到的数量，更新的数量
func (c *mgoClient) UpdateResult(tb string, query, update interface{}) (int64, int64, error) {
	now := time.Now()
	defer func() {
		log.Debugf("[MgoUpdateResult][%v][%s]\n", time.Since(now), tb)
	}()
	cl := c.client.Database(c.db).Collection(tb)
	r, err := cl.UpdateOne(c.ctx, query, update)
	if err != nil {
		return 0, 0, err
	}
	return r.MatchedCount, r.ModifiedCount, err
}

// Upsert 更新或插入，返回true表示更新
func (c *mgoClient) Upsert(tb string, query, update interface{}) (bool, error) {
	now := time.Now()
	defer func() {
		log.Debugf("[MgoUpsert][%v][%s]\n", time.Since(now), tb)
	}()
	cl := c.client.Database(c.db).Collection(tb)
	upsert := true
	r, err := cl.UpdateOne(context.Background(), query, update, &options.UpdateOptions{
		Upsert: &upsert,
	})
	if err != nil {
		return false, err
	}
	return r.MatchedCount == 0, err
}

// UpdateMany 更新多个记录 返回 修改数量,匹配数量,错误
func (c *mgoClient) UpdateMany(tb string, query, update interface{}) (int64, int64, error) {
	now := time.Now()
	defer func() {
		log.Debugf("[MgoUpdateMany][%v][%s]\n", time.Since(now), tb)
	}()
	cl := c.client.Database(c.db).Collection(tb)
	r, err := cl.UpdateMany(c.ctx, query, update)
	if err != nil {
		return 0, 0, err
	}
	return r.ModifiedCount, r.MatchedCount, nil
}

// Get Get
func (c *mgoClient) Get(tb string, query, obj interface{}) error {
	now := time.Now()
	defer func() {
		log.Debugf("[MgoGet][%v][%s]\n", time.Since(now), tb)
	}()
	cl := c.client.Database(c.db).Collection(tb)
	return cl.FindOne(c.ctx, query).Decode(obj)
}

// Count Count
func (c *mgoClient) Count(tb string, query interface{}) (int64, error) {
	now := time.Now()
	defer func() {
		log.Debugf("[MgoCount][%v][%s]\n", time.Since(now), tb)
	}()
	return c.client.Database(c.db).Collection(tb).CountDocuments(c.ctx, query)
}

// Find Find
func (c *mgoClient) Find(tb string, query interface{}, skip, limit int64, sort string, total bool, obj interface{}) (int64, error) {
	now := time.Now()
	defer func() {
		log.Debugf("[MgoFind][%v][%s][%d:%d][%s]\n", time.Since(now), tb, skip, limit, sort)
	}()
	t := int64(0)
	rV := reflect.ValueOf(obj)
	if rV.Elem().Kind() != reflect.Slice {
		return t, ErrObjNotArray
	}

	cl := c.client.Database(c.db).Collection(tb)
	if total {
		tt, err := cl.CountDocuments(c.ctx, query)
		if err != nil {
			return t, err
		}
		t = tt
	}
	opt := options.Find()
	if skip > -1 {
		opt.Skip = &skip
	}
	if limit > -1 {
		opt.Limit = &limit
	}
	if sort != "" {
		sorts := strings.Split(sort, ",")
		sm := map[string]interface{}{}
		for _, v := range sorts {
			if v == "" {
				continue
			}
			if strings.HasPrefix(v, "-") {
				v = strings.Trim(v, "-")
				sm[v] = -1
			} else {
				sm[v] = 1
			}

		}
		if len(sm) > 0 {
			opt.Sort = sm
		}
	}

	cur, err := cl.Find(c.ctx, query, opt)
	if err != nil {
		return t, err
	}

	defer cur.Close(context.Background())
	slicev := rV.Elem()
	slicev = slicev.Slice(0, slicev.Cap())
	elemt := slicev.Type().Elem()
	i := 0
	for cur.Next(context.Background()) {
		if slicev.Len() == i {
			elemp := reflect.New(elemt)
			err := cur.Decode(elemp.Interface())
			if err != nil {
				log.Fatal(err)
			}
			slicev = reflect.Append(slicev, elemp.Elem())
			slicev = slicev.Slice(0, slicev.Cap())
		} else {
			err := cur.Decode(slicev.Index(i).Addr().Interface())
			if err != nil {
				log.Fatal(err)
			}
		}
		i++
	}
	if err := cur.Err(); err != nil {
		log.Fatal(err)
	}
	rV.Elem().Set(slicev.Slice(0, i))
	return t, nil

}

// Del Del
func (c *mgoClient) Del(tb string, query interface{}) error {
	log.Debugf("[MgoDel][%s]\n", tb)
	return c.Update(tb, query, M{"$set": M{"deltime": time.Now().Unix()}})
}

// DelMany DelMany
func (c *mgoClient) DelMany(tb string, query interface{}) (int64, int64, error) {
	log.Debugf("[MgoDel][%s][%s]\n", tb)
	return c.UpdateMany(tb, query, M{"$set": M{"deltime": time.Now().Unix()}})
}
