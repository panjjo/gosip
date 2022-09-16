package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type QueryFilters struct {
	FieldName string           `json:"field_name" description:"字段名"`
	Opertator string           `json:"opertator" description:"条件"`
	Value     interface{}      `json:"value" description:"值"`
	Or        [][]QueryFilters `json:"or" description:"or条件"`
}

type complex struct {
	field       string
	name        string
	table       string
	association string
	query       map[string]interface{}
	fun         func(querys []QueryFilters, pks ...string) (interface{}, error)
}

func (c *complex) Query(querys []QueryFilters, pks ...string) (interface{}, error) {
	if c.fun == nil {
		qms, err := GenQueryMap(querys, pks...)
		if err != nil {
			return "", err
		}
		for k, v := range c.query {
			qms.Where[k] = v
		}
		if c.association != "" {
			switch len(pks) {
			case 0:
			case 1:
				qms.Where[fmt.Sprintf("%s = ?", c.association)] = pks[0]
			default:
				qms.Where[fmt.Sprintf("%s in ('%s')", c.association, strings.Join(pks, "','"))] = nil
			}
		}

		return fmt.Sprintf("SELECT %s from %s where %s and deltime is null", c.field, c.table, qms.SQL()), nil

	} else {
		return c.fun(querys, pks...)
	}
}

var complexFilter = map[string]*complex{}

func FilterRegister(field, name, table, association string, query map[string]interface{}) {
	FilterRegisterFn(field, name, table, association, query, nil)
}

func FilterRegisterFn(field, name, table, association string, query map[string]interface{}, fn func(querys []QueryFilters, pks ...string) (interface{}, error)) {
	complexFilter[name] = &complex{
		field:       field,
		name:        name,
		table:       table,
		association: association,
		query:       query,
		fun:         fn,
	}
}

type QueryMap struct {
	Where map[string]interface{}
	Or    [][]map[string]interface{}
}

func vtosql(v interface{}) string {
	switch t := v.(type) {
	case []string:
		return "'" + strings.Join(t, "','") + "'"
	case []int:
		s := []string{}
		for _, vs := range t {
			s = append(s, fmt.Sprint(vs))
		}
		return strings.Join(s, ",")
	case []int32:
		s := []string{}
		for _, vs := range t {
			s = append(s, fmt.Sprint(vs))
		}
		return strings.Join(s, ",")
	case []int64:
		s := []string{}
		for _, vs := range t {
			s = append(s, fmt.Sprint(vs))
		}
		return strings.Join(s, ",")
	case []float32:
		s := []string{}
		for _, vs := range t {
			s = append(s, fmt.Sprint(vs))
		}
		return strings.Join(s, ",")
	case []float64:
		s := []string{}
		for _, vs := range t {
			s = append(s, fmt.Sprint(vs))
		}
		return strings.Join(s, ",")
	case []bool:
		s := []string{}
		for _, vs := range t {
			s = append(s, fmt.Sprint(vs))
		}
		return strings.Join(s, ",")
	case []interface{}:
		s := []string{}
		for _, vs := range t {
			s = append(s, fmt.Sprint(vs))
		}
		str := false
		if len(t) > 0 {
			switch t[0].(type) {
			case string:
				str = true

			}
		}
		if str {
			return "'" + strings.Join(s, "','") + "'"
		}
		return strings.Join(s, ",")
	}

	return "'" + fmt.Sprint(v) + "'"
}

func (qm QueryMap) SQL() string {

	ws := []string{}

	for k, v := range qm.Where {
		if k != "" {
			ws = append(ws, strings.Replace(k, "?", vtosql(v), 1))
		}
	}
	if qm.Or == nil || len(qm.Or) == 0 {
		return strings.Join(ws, " and ")
	}

	os := []string{}
	for _, or := range qm.Or {
		ows := []string{}
		for _, v := range or {
			for k, v := range v {
				if k != "" {
					ows = append(ows, strings.Replace(k, "?", vtosql(v), 1))
				}
			}
		}
		if len(ows) > 0 {
			os = append(os, strings.Join(ows, " and "))
		}
	}
	wss := strings.Join(ws, " and ")
	oss := "(" + strings.Join(os, ") or (") + ")"

	if len(ws) > 0 {
		return fmt.Sprintf("(%s) and %s", wss, oss)
	} else {
		return oss
	}

}

var (
	ErrJSONEMPTY = errors.New("json data is empty")
)

// GenQueryMapWithJSON 根据前端filters转换成后端sql查询
func GenQueryMapWithJSON(jsondata string, pks ...string) (QueryMap, error) {

	filters := []QueryFilters{}
	if jsondata == "" {
		return QueryMap{
			Where: map[string]interface{}{},
			Or:    [][]map[string]interface{}{},
		}, ErrJSONEMPTY
	}

	d := json.NewDecoder(strings.NewReader(jsondata))
	d.UseNumber()
	err := d.Decode(&filters)
	if err != nil {
		return QueryMap{
			Where: map[string]interface{}{},
			Or:    [][]map[string]interface{}{},
		}, ErrJSONEMPTY
	}
	// fmt.Println("JSONDATA:", filters)
	return GenQueryMap(filters, pks...)
}

func GenQueryMap(querys []QueryFilters, pks ...string) (QueryMap, error) {
	qm := QueryMap{
		Where: map[string]interface{}{},
		Or:    [][]map[string]interface{}{},
	}

	needids := map[string][]QueryFilters{} //需要获取id的字段
	for _, query := range querys {
		if query.FieldName != "" {
			if fn := strings.SplitN(query.FieldName, ".", 2); len(fn) > 1 {
				if _, ok := complexFilter[fn[0]]; ok {
					query.FieldName = fn[1]
					if qs, ok := needids[fn[0]]; ok {
						needids[fn[0]] = append(qs, query)
					} else {
						needids[fn[0]] = []QueryFilters{query}
					}
				}
			} else {
				sql, value, err := filterToSql(query)
				if err != nil {
					return qm, err
				}
				if value != nil {
					qm.Where[sql] = value
				}
			}
		}
		// 处理or
		if len(query.Or) > 0 {
			orm := []map[string]interface{}{}
			for _, o := range query.Or {
				oqm, err := GenQueryMap(o, pks...)
				if err != nil {
					return qm, err
				}
				orm = append(orm, map[string]interface{}{
					oqm.SQL(): nil,
				})
			}
			if len(orm) > 0 {
				qm.Or = append(qm.Or, orm)
			}
		}

	}

	//fmt.Println("NeedIDS:\n", JSONEncodeIndent(needids))
	for k, qs := range needids {
		if fn, ok := complexFilter[k]; ok {
			ids, err := fn.Query(qs, pks...)
			if err != nil {
				return qm, err
			}
			qm.Where[fmt.Sprintf("%v = any (array(%v))", fn.field, ids)] = nil
		}
	}

	return qm, nil
}

var fieldmap = map[string]string{}

func FieldMap(f map[string]string) {
	for k, v := range f {
		fieldmap[k] = v
	}
}

func filterToSql(data QueryFilters) (sql string, value interface{}, err error) {
	filed := data.FieldName

	if strings.Contains(data.FieldName, "'") {
		err = fmt.Errorf("field:%v error", data.FieldName)
	}
	if nf, ok := fieldmap[filed]; ok {
		filed = nf
	}

	switch data.Opertator {
	case "=":
		sql = fmt.Sprintf(`%v = ?`, filed)
		value = data.Value
	case ">":
		sql = fmt.Sprintf(`%v > ?`, filed)
		value = data.Value
	case "<":
		sql = fmt.Sprintf(`%v < ?`, filed)
		value = data.Value
	case ">=":
		sql = fmt.Sprintf(`%v >= ?`, filed)
		value = data.Value
	case "<=":
		sql = fmt.Sprintf(`%v <= ?`, filed)
		value = data.Value
	case "<>":
		sql = fmt.Sprintf(`%v <> ?`, filed)
		value = data.Value
	case "in":
		sql = fmt.Sprintf(`%v in (?)`, filed)
		switch t := data.Value.(type) {
		case string:
			value = strings.Split(strings.Trim(t, ","), ",")
		default:
			value = t
		}
	case "notin":
		sql = fmt.Sprintf(`%v not in (?)`, filed)
		value = data.Value
	case "like":
		sql = fmt.Sprintf(`%v like ?`, filed)
		value = fmt.Sprintf("%%%v%%", data.Value)
	default:
		err = fmt.Errorf("unsupported opertator:%v", data.Opertator)

	}

	return
}
