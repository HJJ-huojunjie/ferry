/*
  @Author : lanyulei
*/

package jsonTime

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

// 重写MarshalJSON实现models json返回的时间格式
type JSONTime struct {
	time.Time
}

func (t JSONTime) MarshalJSON() ([]byte, error) {
	formatted := fmt.Sprintf("\"%s\"", t.Format("2006-01-02 15:04:05"))
	return []byte(formatted), nil
}

// UnmarshalJSON 反序列化，兼容多种时间字符串格式及空值
func (t *JSONTime) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), "\"")
	if s == "" || s == "null" || s == "0001-01-01 00:00:00" {
		*t = JSONTime{Time: time.Time{}}
		return nil
	}
	layouts := []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	var lastErr error
	loc := time.Local
	for _, layout := range layouts {
		parsed, err := time.ParseInLocation(layout, s, loc)
		if err == nil {
			*t = JSONTime{Time: parsed}
			return nil
		}
		lastErr = err
	}
	return fmt.Errorf("无法解析时间 %q: %v", s, lastErr)
}

func (t JSONTime) Value() (driver.Value, error) {
	var zeroTime time.Time
	if t.Time.UnixNano() == zeroTime.UnixNano() {
		return nil, nil
	}
	return t.Time, nil
}

func (t *JSONTime) Scan(v interface{}) error {
	value, ok := v.(time.Time)
	if ok {
		*t = JSONTime{Time: value}
		return nil
	}
	return fmt.Errorf("无法转换 %v 的时间格式", v)
}
