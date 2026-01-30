package utils

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// StructToFlatString 递归转换结构体为单行字符串 用于日志打印
func StructToFlatString(obj any) string {
	return convertValue(reflect.ValueOf(obj), "", 0)
}

func convertValue(v reflect.Value, path string, depth int) string {
	// 防止无限递归，设置最大深度
	if depth > 10 {
		return path + ":<max_depth_reached>"
	}

	// 处理指针类型
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return path + ":nil"
		}
		v = v.Elem()
	}

	// 处理不可导出的字段或零值
	if !v.IsValid() {
		return path + ":<invalid>"
	}

	switch v.Kind() {
	case reflect.Struct:
		// 特殊处理time.Time类型
		if v.Type() == reflect.TypeOf(time.Time{}) {
			return path + ":" + v.Interface().(time.Time).Format("2006-01-02 15:04:05")
		}

		var parts []string
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			// 跳过非导出字段
			if field.PkgPath != "" && !field.Anonymous {
				continue
			}

			fieldValue := v.Field(i)
			fieldName := field.Name

			var newPath string
			if path == "" {
				newPath = fieldName
			} else {
				newPath = path + "." + fieldName
			}

			part := convertValue(fieldValue, newPath, depth+1)
			parts = append(parts, part)
		}
		return strings.Join(parts, " ")

	case reflect.Slice, reflect.Array:
		if v.Len() == 0 {
			return path + ":[]"
		}

		var parts []string
		for i := 0; i < v.Len(); i++ {
			elemValue := v.Index(i)
			newPath := path + "[" + strconv.Itoa(i) + "]"
			part := convertValue(elemValue, newPath, depth+1)
			parts = append(parts, part)
		}
		return strings.Join(parts, " ")

	case reflect.Map:
		if v.Len() == 0 {
			return path + ":{}"
		}

		var parts []string
		for _, key := range v.MapKeys() {
			mapValue := v.MapIndex(key)
			keyStr := fmt.Sprintf("%v", key.Interface())
			// 处理key中的特殊字符
			keyStr = strings.ReplaceAll(keyStr, " ", "_")
			newPath := path + "[" + keyStr + "]"
			part := convertValue(mapValue, newPath, depth+1)
			parts = append(parts, part)
		}
		return strings.Join(parts, " ")

	case reflect.String:
		return path + ":\"" + v.String() + "\""

	case reflect.Bool:
		return path + ":" + strconv.FormatBool(v.Bool())

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return path + ":" + strconv.FormatInt(v.Int(), 10)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return path + ":" + strconv.FormatUint(v.Uint(), 10)

	case reflect.Float32, reflect.Float64:
		return path + ":" + strconv.FormatFloat(v.Float(), 'f', -1, 64)

	case reflect.Interface:
		return convertValue(v.Elem(), path, depth+1)

	default:
		return path + ":" + fmt.Sprintf("%v", v.Interface())
	}
}

// Contains 是否含有元素
func Contains[T comparable](s []T, i T) bool {
	for _, v := range s {
		if v == i {
			return true
		}
	}

	return false
}

// GetTimestamp 获取毫秒级时间戳
func GetTimestamp() int64 {
	now := time.Now()
	milliseconds := now.UnixNano() / int64(time.Millisecond)
	return milliseconds
}

// Base64Encode 对字符串进行Base64编码
func Base64Encode(str string) string {
	// 将字符串转换为字节切片
	data := []byte(str)
	// 使用标准Base64编码
	return base64.RawURLEncoding.EncodeToString(data)
}

// Base64Decode 对Base64字符串进行解码
func Base64Decode(encodedStr string) (string, error) {
	// 使用标准Base64解码
	data, err := base64.RawURLEncoding.DecodeString(encodedStr)
	if err != nil {
		return "", fmt.Errorf("base64解码失败: %v", err)
	}
	return string(data), nil
}

// RandomString 生成指定长度的随机字符串
func RandomString(length int) string {
	// 初始化随机种子
	rand.NewSource(time.Now().UnixNano())

	const letters = "abcdefghijklmnopqrstuvwxyz"

	b := make([]byte, length)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// RemoveItem 移除切片中的指定元素
func RemoveItem[T comparable](slice []T, target T) []T {
	result := make([]T, 0, len(slice))
	for _, v := range slice {
		if v != target {
			result = append(result, v)
		}
	}
	return result
}

// ReverseSlice 将切片的元素倒序排列
func ReverseSlice[T any](s []T) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

// RemoveDuplicates 删除切片中重复的元素
func RemoveDuplicates[T comparable](slice []T) []T {
	seen := make(map[T]struct{})
	result := make([]T, 0, len(slice))

	for _, v := range slice {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}

	return result
}
