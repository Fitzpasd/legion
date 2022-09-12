package main

import (
	"bytes"
	"reflect"
	"testing"
)

func testHelper(actual, expected []byte, t *testing.T) {
	if !bytes.Equal(actual, expected) {
		t.Errorf("\nActual:{[% x]}\nExpected:{[% x]}", actual, expected)
	}
}

func encodeAndIgnoreError(data any) []byte {
	encoded, _ := Encode(data)
	return encoded
}

func decodeAndIgnoreError(data []byte) any {
	decoded, _ := Decode(data)
	return decoded
}

func TestEncodeEmptyString(t *testing.T) {
	var asString string = ""
	var asInterface interface{} = ""
	expected := []byte{0x80}

	testHelper(encodeAndIgnoreError(asString), expected, t)
	testHelper(encodeAndIgnoreError(asInterface), expected, t)
}

func TestEncodeNonEmptyString(t *testing.T) {
	testHelper(encodeAndIgnoreError("dog"), []byte{0x83, 'd', 'o', 'g'}, t)
	testHelper(encodeAndIgnoreError([]byte("dog")), []byte{0x83, 'd', 'o', 'g'}, t)
	testHelper(
		encodeAndIgnoreError("Lorem ipsum dolor sit amet, consectetur adipisicing elit"),
		[]byte{0xb8, 0x38, 'L', 'o', 'r', 'e', 'm', ' ', 'i', 'p', 's', 'u',
			'm', ' ', 'd', 'o', 'l', 'o', 'r', ' ', 's', 'i', 't', ' ', 'a',
			'm', 'e', 't', ',', ' ', 'c', 'o', 'n', 's', 'e', 'c', 't', 'e',
			't', 'u', 'r', ' ', 'a', 'd', 'i', 'p', 'i', 's', 'i', 'c', 'i',
			'n', 'g', ' ', 'e', 'l', 'i', 't'}, t)
}

func TestEncodeStringList(t *testing.T) {
	actual := encodeAndIgnoreError([]string{"cat", "dog"})
	expected := []byte{0xc8, 0x83, 'c', 'a', 't', 0x83, 'd', 'o', 'g'}
	testHelper(actual, expected, t)
}

func TestEncodeEmptyList(t *testing.T) {
	expected := []byte{0xc0}
	testHelper(encodeAndIgnoreError([]string{}), expected, t)
	testHelper(encodeAndIgnoreError([]byte{}), expected, t)
	testHelper(encodeAndIgnoreError([]interface{}{}), expected, t)
}

func TestEncodeSetTheoreticalRepresentation(t *testing.T) {
	// [ [], [[]], [ [], [[]] ] ]
	actual := encodeAndIgnoreError([]interface{}{
		[]interface{}{},
		[][]interface{}{{}},
		[]interface{}{[]string{}, [][]interface{}{{}}},
	})

	expected := []byte{0xc7, 0xc0, 0xc1, 0xc0, 0xc3, 0xc0, 0xc1, 0xc0}

	testHelper(actual, expected, t)
}

func TestEncodeInteger(t *testing.T) {
	testHelper(encodeAndIgnoreError(0), []byte{0x80}, t)
	testHelper(encodeAndIgnoreError(1), []byte{0x01}, t)
	testHelper(encodeAndIgnoreError(15), []byte{0x0f}, t)
	testHelper(encodeAndIgnoreError(1024), []byte{0x82, 0x04, 0x00}, t)
	testHelper(encodeAndIgnoreError(30303), []byte{0x82, 0x76, 0x5f}, t)
}

func TestEncodeByteArray(t *testing.T) {
	testHelper(encodeAndIgnoreError([]byte{127, 0, 0, 1}), []byte{0x84, 0x7f, 0, 0, 1}, t)
}

func TestDecodeEmptyString(t *testing.T) {
	if decodeAndIgnoreError([]byte{0x80}) != "" {
		t.Error("Failed to decode empty string")
	}
}

func TestDecodeString(t *testing.T) {
	if decodeAndIgnoreError([]byte{0x83, 'd', 'o', 'g'}) != "dog" {
		t.Error("Failed to decode 'dog'")
	}

	actual := decodeAndIgnoreError([]byte{0xb8, 0x38, 'L', 'o', 'r', 'e', 'm', ' ', 'i', 'p', 's', 'u',
		'm', ' ', 'd', 'o', 'l', 'o', 'r', ' ', 's', 'i', 't', ' ', 'a',
		'm', 'e', 't', ',', ' ', 'c', 'o', 'n', 's', 'e', 'c', 't', 'e',
		't', 'u', 'r', ' ', 'a', 'd', 'i', 'p', 'i', 's', 'i', 'c', 'i',
		'n', 'g', ' ', 'e', 'l', 'i', 't'})

	expected := "Lorem ipsum dolor sit amet, consectetur adipisicing elit"

	if actual != expected {
		t.Error("Failed to decode Lorem ipsum")
	}
}

func TestDecodeStringList(t *testing.T) {
	actual := decodeAndIgnoreError([]byte{0xc8, 0x83, 'c', 'a', 't', 0x83, 'd', 'o', 'g'})
	expected := []any{"cat", "dog"}

	if !reflect.DeepEqual(actual, expected) {
		t.Error("Failed to decode string list")
	}
}

func TestDecodeEmptyList(t *testing.T) {
	actual := decodeAndIgnoreError([]byte{0xc0})
	rt := reflect.ValueOf(actual)

	if !(rt.Kind() == reflect.Array || rt.Kind() == reflect.Slice) && rt.Len() != 0 {
		t.Error("Failed to decode empty list")
	}
}

func isArrayOrSlice(v reflect.Value) bool {
	return v.Kind() == reflect.Array || v.Kind() == reflect.Slice
}

func checkLength(v reflect.Value, i, l int) bool {
	return v.Index(i).Elem().Len() == l
}

func TestDecodeSetTheoreticalRepresentation(t *testing.T) {
	// [ [], [[]], [ [], [[]] ] ]
	actual := decodeAndIgnoreError([]byte{0xc7, 0xc0, 0xc1, 0xc0, 0xc3, 0xc0, 0xc1, 0xc0})
	rt := reflect.ValueOf(actual)

	isValid := isArrayOrSlice(rt) && rt.Len() == 3 &&
		// Verify size of top level elements
		isArrayOrSlice(rt.Index(0).Elem()) && checkLength(rt, 0, 0) &&
		isArrayOrSlice(rt.Index(1).Elem()) && checkLength(rt, 1, 1) &&
		isArrayOrSlice(rt.Index(2).Elem()) && checkLength(rt, 2, 2) &&

		// Verify second index item
		isArrayOrSlice(rt.Index(1).Elem().Index(0).Elem()) && checkLength(rt.Index(1).Elem(), 0, 0) &&

		// Verify third index item
		isArrayOrSlice(rt.Index(2).Elem().Index(0).Elem()) && checkLength(rt.Index(2).Elem(), 0, 0) &&
		isArrayOrSlice(rt.Index(2).Elem().Index(1).Elem()) && checkLength(rt.Index(2).Elem(), 1, 1) &&
		isArrayOrSlice(rt.Index(2).Elem().Index(1).Elem().Index(0).Elem()) && checkLength(rt.Index(2).Elem().Index(1).Elem(), 0, 0)

	if !isValid {
		t.Error("Failed to decode set theoretical representation")
	}
}
