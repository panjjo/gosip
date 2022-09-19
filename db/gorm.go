package db

import (
	"reflect"
	"strings"
	"time"

	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/panjjo/gorm"
	//  _ "github.com/jinzhu/gorm/dialects/postgres"
	//  _ "github.com/jinzhu/gorm/dialects/sqlite"
	//  _ "github.com/jinzhu/gorm/dialects/mssql"
)

type Config struct {
	Dialect string `json:"dialect" yaml:"dialect"`
	URL     string `json:"url" yaml:"url"`
}

var DBClient *gorm.DB

func Open(config Config) (*gorm.DB, error) {
	return gorm.Open(config.Dialect, config.URL)
}

func KeepLive(db *gorm.DB, d time.Duration) {
	defer func() {
		go KeepLive(db, d)
	}()

	for {
		db.DB().Ping()
		time.Sleep(d)
	}
}

func Create(db *gorm.DB, obj any) error {
	return db.Create(obj).Error
}

func Save(db *gorm.DB, obj any) error {
	return db.Save(obj).Error
}

func UpdateAll(db *gorm.DB, model any, query map[string]any, update any) (int64, error) {
	db = db.Model(model)
	for k, v := range query {
		db = db.Where(k, v)
	}
	db = db.Updates(update)
	return db.RowsAffected, db.Error
}

func Get(db *gorm.DB, obj any) error {
	return db.Where(obj).First(obj).Error
}
func GetQ(db *gorm.DB, obj any, query map[string]any, ors ...[]map[string]any) error {
	t := GenQueryDB(db, query, ors...)
	return t.First(obj).Error
}

func Del(db *gorm.DB, obj any) error {
	return db.Where(obj).Delete(obj).Error
}
func DelQ(db *gorm.DB, obj any, query map[string]any, ors ...[]map[string]any) error {
	t := GenQueryDB(db, query, ors...)
	return t.Delete(obj).Error
}
func Find(db *gorm.DB, query map[string]any, ors [][]map[string]any, sort string, skip, limit int, objs any) error {
	ndb := GenQueryDB(db, query, ors...)
	ndb = OLO(ndb, sort, skip, limit)
	return ndb.Find(objs).Error
}
func FindT(db *gorm.DB, model any, objs any, query map[string]any, order string, skip, limit int, total bool) (int64, error) {
	db = GenQueryDB(db, query)
	db = OLO(db, order, skip, limit)
	return Count(db, model, total), db.Find(objs).Error
}

func FindWithJson(db *gorm.DB, model any, objs any, query, order string, skip, limit int, total bool) (int64, error) {
	if query != "" {
		qmap, err := GenQueryMapWithJSON(query)
		if err != nil {
			return 0, err
		}
		db = GenQueryDB(db, qmap.Where, qmap.Or...)
	}
	db = OLO(db, order, skip, limit)

	return Count(db, model, total), db.Find(objs).Error
}

func Count(db *gorm.DB, m any, reqtotal bool) int64 {
	total := int64(0)
	if reqtotal {
		db.Model(m).Offset(-1).Limit(-1).Count(&total)
	}
	return total
}

// DBPreload 数据根据字段预加载时添加扩展,preloads value 支持单项数据以及 []interface{}{"Field", ...} 数据
func Preload(db *gorm.DB, fields map[string]struct{}, preloads map[string]any) *gorm.DB {
	for k, v := range preloads {
		if _, ok := fields[k]; ok {
			vT := reflect.TypeOf(v)
			switch vT.Kind() {
			case reflect.Array, reflect.Slice:
				nv, ok := v.([]any)
				if !ok {
					continue
				}
				db = db.Preload(nv[0].(string), nv[1:]...)
			case reflect.String:
				db = db.Preload(v.(string))
			default:
			}
		}
	}
	return db
}
func GenQueryDB(db *gorm.DB, Where map[string]any, Ors ...[]map[string]any) *gorm.DB {
	for k, v := range Where {
		if len(k) == 0 {
			continue
		}
		if v != nil {
			if strings.Count(k, "?") > 1 {
				if nv, ok := v.([]any); ok {
					db = db.Where(k, nv...)
				}
			} else {
				db = db.Where(k, v)
			}
		} else {
			db = db.Where(k)
		}
	}
	if len(Ors) > 0 {
		s, v := GenOr(Ors...)
		if len(s) > 0 {
			db = db.Where(s, v...)
		}
	}
	return db
}

// GenOr 生成or的sql
func GenOr(Ors ...[]map[string]any) (string, []any) {
	sqlstr := []string{}
	sqlV := []any{}
	for _, or := range Ors {
		if len(or) == 0 {
			continue
		}
		ostr := []string{}
		for _, o := range or {
			if len(o) == 0 {
				continue
			}
			astr := []string{}
			for sql, value := range o {
				astr = append(astr, sql)
				if strings.Count(sql, "?") > 0 {
					sqlV = append(sqlV, value)
				}
			}
			if len(astr) > 0 {
				ostr = append(ostr, strings.Join(astr, " AND "))
			}
		}
		if len(ostr) > 0 {
			sqlstr = append(sqlstr, "("+strings.Join(ostr, ") OR (")+")")
		}
	}

	return "(" + strings.Join(sqlstr, ") AND (") + ")", sqlV
}
func OLO(db *gorm.DB, sort string, offset, limit int) *gorm.DB {
	sort = strings.Trim(sort, ",")
	if sort != "" {
		orders := strings.Split(sort, ",")
		for _, o := range orders {
			n := strings.TrimSpace(o)
			if strings.HasPrefix(n, "-") {
				db = db.Order(strings.Trim(n, "-") + " desc")
			} else {

				db = db.Order(o)
			}
		}
	}
	if offset > -1 {
		db = db.Offset(offset)
	}
	if limit > -1 {
		db = db.Limit(limit)
	}
	return db
}

// Fields 给db添加Select
func Fields(db *gorm.DB, fields map[string]struct{}) *gorm.DB {
	fs := []string{}
	for k := range fields {
		if !strings.Contains(k, ".") {
			fs = append(fs, k)
		}
	}
	return db.Select(fs)
}
