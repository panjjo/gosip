package db

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/panjjo/gorm"
	"github.com/panjjo/gosip/utils"
)

type DBModel struct {
	ID        uint   `json:"id" gorm:"primary_key"`
	CreatedAt int64  `json:"addtime" gorm:"column:addtime"`
	UpdatedAt int64  `json:"uptime" gorm:"column:uptime"`
	DeletedAt *int64 `json:"-" sql:"index" gorm:"column:deltime"`
}

type M map[string]interface{}

func (j M) Value() (driver.Value, error) {
	return utils.JSONEncode(&j), nil
}

func (j *M) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
	}

	return utils.JSONDecode(bytes, j)
}

type StringArray []string

func (j StringArray) Value() (driver.Value, error) {
	return strings.Join(j, ","), nil
}

func (j *StringArray) Scan(value interface{}) error {
	switch t := value.(type) {
	case []byte:
		nv := StringArray(strings.Split(string(t), ","))
		*j = nv
	case string:
		nv := StringArray(strings.Split(t, ","))
		*j = nv
	}
	return nil
}

type StringArrayJSON []string

func (j StringArrayJSON) Value() (driver.Value, error) {
	return utils.JSONEncode(&j), nil
}

func (j *StringArrayJSON) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
	}

	return utils.JSONDecode(bytes, j)
}

type Int64Array []int64

func (j Int64Array) Value() (driver.Value, error) {
	return strings.Join(its(j), ","), nil
}

func (j *Int64Array) Scan(value interface{}) error {
	switch t := value.(type) {
	case []byte:
		nv := strings.Split(string(t), ",")
		is := Int64Array(sti(nv))
		*j = is
	case string:
		nv := strings.Split(t, ",")
		is := Int64Array(sti(nv))
		*j = is
	}
	return nil
}

type Int64ArrayJSON []int64

func (j Int64ArrayJSON) Value() (driver.Value, error) {
	return utils.JSONEncode(&j), nil
}

func (j *Int64ArrayJSON) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
	}

	return utils.JSONDecode(bytes, j)
}
func its(is []int64) []string {
	ss := make([]string, len(is))
	for i, v := range is {
		ss[i] = strconv.FormatInt(v, 10)
	}
	return ss
}
func sti(ss []string) []int64 {
	is := make([]int64, len(ss))
	for i, v := range ss {
		is[i], _ = strconv.ParseInt(v, 10, 64)
	}
	return is
}

func RecordNotFound(e error) bool {
	return errors.Is(e, gorm.ErrRecordNotFound)
}
