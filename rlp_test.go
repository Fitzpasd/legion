package main

import (
	"bytes"
	"encoding/hex"
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

func TestDecodeFindNeighbors(t *testing.T) {
	response := "f90144f9013cf84d8422c9778882765f82765fb840714e2c3c5fc0e9b1336980b074c2265cce99370003da4a46309e5a524f76e51908093e231eb0be3afc575e79a382dea7c28174e37665ab871fa7eba8831a0c9ff84d844624325182765f82765fb8409a4ed2bd8779a63bc9a4ddbd4aa9c6103b4ba9a90dd222d360a7ccbf9ef10151cf51451741aac3728f2da99810eb643b825334a998ae102c08b5097ec0a955f2f84d84035da20482765f82765fb8408dff7c932a24e042f88b5bec2b92dc39b133c303ce2ac8791f0757fd1bab99088dcd4818d26edccff5c575a25728e80d4b5f237ab4adcd788ce2135af009d736f84d842249a2f38274c08274c0b84087eb3e6eeb3d15fba0a478cf28886154cc7e25b9a914b9c0487cd93ae51eba9ef803bd8734841b0f57141c35144baed8a442b110428291ba672122a8e8aa1cd68463275e95944d82765f82765fb840e5412a9aac4ff02ff253c0df7046620cc92933e5e6416dcbbfc5b5a8d7960e2a8b230f5a9afd1019b0e3b1cacde283f91b87ce158fb81822865b11a4d1fb6026f84d84955ba03882765f82765fb8405d6c602f9cd2e1e8b829fa318f544531262d711abc4e7a7e5e54f35900e927cec177c41ad4207a48295e8c6a50d6b9039a370bbdc6087531b20e6c66ecb3c874f84d848d5f20c182765f82765fb840827c4feb458434749cdfec2d7aa579c127b20875f13b2022ff2381d434e583250d8490406aa498db16e5990a803f4b4a7f321bb5e9177466057fd2cd86bf603ef84d843649e239828214828214b840f6f6e88eec3e9cf40160204ac2e8bdfbf4b8cdef4f8fa0eead83c8a2cd4573daaddc26ea06c7c03cb984ddcfb07521c5805dd97abed8346ce55e164647b0fce4f84d84175805ab827a44827a44b8401578af2568b9be4913b77bf4a585f9720529bde01bacdd83ff742aca3b42680deb1ba864fbfc3463f5842f920a5cc186aab5aaabd7062a2e0e2c183e498e3c73f84d8436f2f08e82765f82765fb840e2fb40a4c274539b7b80d8ca0b84fd3bfb4d7db0647dc2e8ffdba496f403bc162c97d04b59fb3de8257c3684b08240bc01b35008966158854ae5c241d6be0f66f84d846222d2f582765f82765fb840df2579a9bb9dbf6d492c831a843433c1c5f68850682bac8c3e4c0cbbee84697800c56efd45ebee32907f8cb685c7eebb075d1f1bd659ccfd773a6eaf96d9ef18f84d840359cd4b82765f82765fb840e955154bb5370276118d3b54d75280a0f738cdbe98754901a410e8f2ff6e9283944fa9ec2449b71ead9ede47485aec91f5baa018550da5f72bf177c46d8a20178463275e95"
	responseBytes, _ := hex.DecodeString(response)

	decoded, err := Decode(responseBytes)

	if err != nil {
		t.Error(err)
	}

	nodes := decoded.([]any)[0].([]any)

	expected := [4][4]string{
		{"\"\xc9w\x88", "v_", "v_", "qN,<_\xc0\xe9\xb13i\x80\xb0t\xc2&\\Ι7\x00\x03\xdaJF0\x9eZROv\xe5\x19\b\t>#\x1e\xb0\xbe:\xfcW^y\xa3\x82ާ\u0081t\xe3ve\xab\x87\x1f\xa7먃\x1a\f\x9f"},
		{"F$2Q", "v_", "v_", "\x9aNҽ\x87y\xa6;ɤݽJ\xa9\xc6\x10;K\xa9\xa9\r\xd2\"\xd3`\xa7̿\x9e\xf1\x01Q\xcfQE\x17A\xaa\xc3r\x8f-\xa9\x98\x10\xebd;\x82S4\xa9\x98\xae\x10,\b\xb5\t~\xc0\xa9U\xf2"},
		{"\x03]\xa2\x04", "v_", "v_", "\x8d\xff|\x93*$\xe0B\xf8\x8b[\xec+\x92\xdc9\xb13\xc3\x03\xce*\xc8y\x1f\aW\xfd\x1b\xab\x99\b\x8d\xcdH\x18\xd2n\xdc\xcf\xf5\xc5u\xa2W(\xe8\rK_#z\xb4\xad\xcdx\x8c\xe2\x13Z\xf0\t\xd76"},
		{"\"I\xa2\xf3", "t\xc0", "t\xc0", "\x87\xeb>n\xeb=\x15\xfb\xa0\xa4x\xcf(\x88aT\xcc~%\xb9\xa9\x14\xb9\xc0H|\xd9:\xe5\x1e\xba\x9e\xf8\x03\xbd\x874\x84\x1b\x0fW\x14\x1c5\x14K\xaeؤB\xb1\x10B\x82\x91\xbag!\"\xa8\xe8\xaa\x1c\xd6"},
	}

	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			if nodes[i].([]any)[j].(string) != expected[i][j] {
				t.Error("Unexpected node value")
				return
			}
		}
	}

	expiration := decoded.([]any)[1].(string)

	if expiration != "c'^\x95" {
		t.Error("Unexpected expiration")
	}
}
