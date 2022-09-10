package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
)

func getLengthInBytes(length int) byte {
	binaryLength := len(strconv.FormatInt(int64(length), 2))
	return byte(math.Ceil(float64(binaryLength) / 8))
}

func encodeString(data string) []byte {
	var strLen = len(data)

	var output []byte

	switch {
	case strLen == 1 && data[0] < 0x80:
		return append(output, data[0])

	case strLen < 56:
		output = append(output, 0x80+byte(strLen))
		return append(output, data...)

	default:
		output = append(output, 0xb7+getLengthInBytes(strLen))
		output = append(output, byte(strLen))
		return append(output, data...)
	}
}

func encodeList(data any) ([]byte, error) {
	items := reflect.ValueOf(data)

	if items.Len() == 0 {
		return []byte{0xc0}, nil
	}

	allBytes := true
	for i := 0; i < items.Len(); i++ {
		if _, ok := items.Index(i).Interface().(byte); !ok {
			allBytes = false
			break
		}
	}

	if allBytes {
		var sb strings.Builder

		for i := 0; i < items.Len(); i++ {
			sb.WriteByte(items.Index(i).Interface().(byte))
		}

		return encodeString(sb.String()), nil
	} else {
		var output []byte

		for i := 0; i < items.Len(); i++ {
			item := items.Index(i)
			blah, err := Encode(item.Interface())

			if err != nil {
				return nil, err
			} else {
				output = append(output, blah...)
			}
		}

		var outputLen = len(output)
		switch {
		case outputLen < 56:
			return append([]byte{0xc0 + byte(len(output))}, output...), nil

		default:
			return append([]byte{0xf7 + getLengthInBytes(outputLen), byte(outputLen)}, output...), nil
		}
	}
}

func encodeUInt(i uint) ([]byte, error) {
	switch {
	case i == 0:
		return []byte{0x80}, nil
	case i < 128:
		return []byte{byte(i)}, nil
	default:
		buf := new(bytes.Buffer)
		err := binary.Write(buf, binary.BigEndian, uint32(i))

		if err != nil {
			return nil, err
		} else {
			bytes := buf.Bytes()
			i := 0

			for bytes[i] == 0 && i < len(bytes) {
				i++
			}

			bytes = bytes[i:]

			return append([]byte{0x80 + byte(len(bytes))}, bytes...), nil
		}
	}
}

func Encode(data any) ([]byte, error) {
	isList := false

	switch t := data.(type) {
	case string:
		return encodeString(data.(string)), nil
	case byte:
		return encodeString(string(data.(byte))), nil
	case []byte:
		if len(data.([]byte)) == 0 {
			return []byte{0xc0}, nil
		}
		return encodeString(string(data.([]byte))), nil
	case int:
		return encodeUInt(uint(data.(int)))
	case uint:
		return encodeUInt(data.(uint))

	case []any, []string:
		isList = true

	case any:
		items := reflect.ValueOf(data)
		isList = items.Kind() == reflect.Array || items.Kind() == reflect.Slice

	default:
		return nil, errors.New(fmt.Sprintf("Unsupported type %s", t))
	}

	if isList {
		return encodeList(data)
	} else {
		return nil, errors.New(fmt.Sprintf("Unexpected value %s", data))
	}
}
