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

func encodeUInt(i uint64) ([]byte, error) {
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
		return encodeUInt(uint64(data.(int)))
	case uint:
		return encodeUInt(uint64(data.(uint)))
	case uint64:
		return encodeUInt(data.(uint64))

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

func decodeNextList(data []byte, start int) (any, int, error) {
	prefix := data[start]

	var list []byte

	switch {
	case prefix >= 0xc0 && prefix <= 0xf7:
		listLength := int(prefix - 0xc0)

		if listLength == 0 {
			return []byte{}, start + 1, nil
		}

		end := start + 1 + listLength
		list = data[start+1 : end]

	case prefix >= 0xf8 && prefix <= 0xff:
		listLengthEnd := start + 1 + int(prefix-0xf7)
		listLengthBytes := data[start+1 : listLengthEnd]

		for len(listLengthBytes) != 4 {
			listLengthBytes = append([]byte{0}, listLengthBytes...)
		}

		listLength := int(binary.BigEndian.Uint32(listLengthBytes))
		end := listLengthEnd + listLength
		list = data[listLengthEnd:end]

	default:
		return nil, 0, errors.New(fmt.Sprintf("Unexpected prefix %b", prefix))
	}

	output := []any{}

	i := 0
	for {
		l, n, e := decodeNext(list, i)

		if e != nil {
			return nil, 0, e
		}

		output = append(output, l)

		if n >= len(list) {
			break
		} else {
			i = n
		}
	}

	return output, start + len(list) + 1, nil
}

func decodeNext(data []byte, start int) (any, int, error) {
	prefix := data[start]

	switch {
	case prefix < 0x80:
		return string(prefix), start + 1, nil

	case prefix >= 0x80 && prefix <= 0xb7:
		stringLength := int(prefix - 0x80)
		end := start + 1 + stringLength
		return string(data[start+1 : end]), end, nil

	case prefix >= 0xb8 && prefix <= 0xbf:
		stringLengthEnd := start + 1 + int(prefix-0xb7)
		stringLengthBytes := data[start+1 : stringLengthEnd]

		for len(stringLengthBytes) != 4 {
			stringLengthBytes = append([]byte{0}, stringLengthBytes...)
		}

		stringLength := int(binary.BigEndian.Uint32(stringLengthBytes))
		end := stringLengthEnd + stringLength
		return string(data[stringLengthEnd:end]), end, nil

	case prefix >= 0xc0 && prefix <= 0xff:
		return decodeNextList(data, start)
	default:
		return nil, 0, errors.New(fmt.Sprintf("Unexpected prefix %b", prefix))
	}
}

func Decode(data []byte) (any, error) {
	decoded, _, err := decodeNext(data, 0)
	return decoded, err
}
