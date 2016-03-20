// Copyright (c) 2016, Gerasimos Maropoulos
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice,
//    this list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice,
//	  this list of conditions and the following disclaimer
//    in the documentation and/or other materials provided with the distribution.
//
// 3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse
//    or promote products derived from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL JULIEN SCHMIDT BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
//
// This source code file is based on the Gorilla's sessions package.
//
package sessions

import (
	"crypto/aes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// Asserts that cookieError and MultiError are Error implementations.
var _ Error = cookieError{}
var _ Error = MultiError{}

var testCookies = []interface{}{
	map[string]string{"foo": "bar"},
	map[string]string{"baz": "ding"},
}

var testStrings = []string{"foo", "bar", "baz"}

func TestSecureCookie(t *testing.T) {
	// TODO test too old / too new timestamps
	s1 := NewSecureCookie([]byte("12345"), []byte("1234567890123456"))
	s2 := NewSecureCookie([]byte("54321"), []byte("6543210987654321"))
	value := map[string]interface{}{
		"foo": "bar",
		"baz": 128,
	}

	for i := 0; i < 50; i++ {
		// Running this multiple times to check if any special character
		// breaks encoding/decoding.
		encoded, err1 := s1.Encode("sid", value)
		if err1 != nil {
			t.Error(err1)
			continue
		}
		dst := make(map[string]interface{})
		err2 := s1.Decode("sid", encoded, &dst)
		if err2 != nil {
			t.Fatalf("%v: %v", err2, encoded)
		}
		if !reflect.DeepEqual(dst, value) {
			t.Fatalf("Expected %v, got %v.", value, dst)
		}
		dst2 := make(map[string]interface{})
		err3 := s2.Decode("sid", encoded, &dst2)
		if err3 == nil {
			t.Fatalf("Expected failure decoding.")
		}
		err4, ok := err3.(Error)
		if !ok {
			t.Fatalf("Expected error to implement Error, got: %#v", err3)
		}
		if !err4.IsDecode() {
			t.Fatalf("Expected DecodeError, got: %#v", err4)
		}

		// Test other error type flags.
		if err4.IsUsage() {
			t.Fatalf("Expected IsUsage() == false, got: %#v", err4)
		}
		if err4.IsInternal() {
			t.Fatalf("Expected IsInternal() == false, got: %#v", err4)
		}
	}
}

func TestSecureCookieNilKey(t *testing.T) {
	s1 := NewSecureCookie(nil, nil)
	value := map[string]interface{}{
		"foo": "bar",
		"baz": 128,
	}
	_, err := s1.Encode("sid", value)
	if err != errHashKeyNotSet {
		t.Fatal("Wrong error returned:", err)
	}
}

func TestDecodeInvalid(t *testing.T) {
	// List of invalid cookies, which must not be accepted, base64-decoded
	// (they will be encoded before passing to Decode).
	invalidCookies := []string{
		"",
		" ",
		"\n",
		"||",
		"|||",
		"cookie",
	}
	s := NewSecureCookie([]byte("12345"), nil)
	var dst string
	for i, v := range invalidCookies {
		for _, enc := range []*base64.Encoding{
			base64.StdEncoding,
			base64.URLEncoding,
		} {
			err := s.Decode("name", enc.EncodeToString([]byte(v)), &dst)
			if err == nil {
				t.Fatalf("%d: expected failure decoding", i)
			}
			err2, ok := err.(Error)
			if !ok || !err2.IsDecode() {
				t.Fatalf("%d: Expected IsDecode(), got: %#v", i, err)
			}
		}
	}
}

func TestAuthentication(t *testing.T) {
	hash := hmac.New(sha256.New, []byte("secret-key"))
	for _, value := range testStrings {
		hash.Reset()
		signed := createMac(hash, []byte(value))
		hash.Reset()
		err := verifyMac(hash, []byte(value), signed)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestEncryption(t *testing.T) {
	block, err := aes.NewCipher([]byte("1234567890123456"))
	if err != nil {
		t.Fatalf("Block could not be created")
	}
	var encrypted, decrypted []byte
	for _, value := range testStrings {
		if encrypted, err = encrypt(block, []byte(value)); err != nil {
			t.Error(err)
		} else {
			if decrypted, err = decrypt(block, encrypted); err != nil {
				t.Error(err)
			}
			if string(decrypted) != value {
				t.Errorf("Expected %v, got %v.", value, string(decrypted))
			}
		}
	}
}

func TestGobSerialization(t *testing.T) {
	var (
		sz           GobEncoder
		serialized   []byte
		deserialized map[string]string
		err          error
	)
	for _, value := range testCookies {
		if serialized, err = sz.Serialize(value); err != nil {
			t.Error(err)
		} else {
			deserialized = make(map[string]string)
			if err = sz.Deserialize(serialized, &deserialized); err != nil {
				t.Error(err)
			}
			if fmt.Sprintf("%v", deserialized) != fmt.Sprintf("%v", value) {
				t.Errorf("Expected %v, got %v.", value, deserialized)
			}
		}
	}
}

func TestJSONSerialization(t *testing.T) {
	var (
		sz           JSONEncoder
		serialized   []byte
		deserialized map[string]string
		err          error
	)
	for _, value := range testCookies {
		if serialized, err = sz.Serialize(value); err != nil {
			t.Error(err)
		} else {
			deserialized = make(map[string]string)
			if err = sz.Deserialize(serialized, &deserialized); err != nil {
				t.Error(err)
			}
			if fmt.Sprintf("%v", deserialized) != fmt.Sprintf("%v", value) {
				t.Errorf("Expected %v, got %v.", value, deserialized)
			}
		}
	}
}

func TestEncoding(t *testing.T) {
	for _, value := range testStrings {
		encoded := encode([]byte(value))
		decoded, err := decode(encoded)
		if err != nil {
			t.Error(err)
		} else if string(decoded) != value {
			t.Errorf("Expected %v, got %s.", value, string(decoded))
		}
	}
}

func TestMultiError(t *testing.T) {
	s1, s2 := NewSecureCookie(nil, nil), NewSecureCookie(nil, nil)
	_, err := EncodeMulti("sid", "value", s1, s2)
	if len(err.(MultiError)) != 2 {
		t.Errorf("Expected 2 errors, got %s.", err)
	} else {
		if strings.Index(err.Error(), "hash key is not set") == -1 {
			t.Errorf("Expected missing hash key error, got %s.", err.Error())
		}
		ourErr, ok := err.(Error)
		if !ok || !ourErr.IsUsage() {
			t.Fatalf("Expected error to be a usage error; got %#v", err)
		}
		if ourErr.IsDecode() {
			t.Errorf("Expected error NOT to be a decode error; got %#v", ourErr)
		}
		if ourErr.IsInternal() {
			t.Errorf("Expected error NOT to be an internal error; got %#v", ourErr)
		}
	}
}

func TestMultiNoCodecs(t *testing.T) {
	_, err := EncodeMulti("foo", "bar")
	if err != errNoCodecs {
		t.Errorf("EncodeMulti: bad value for error, got: %v", err)
	}

	var dst []byte
	err = DecodeMulti("foo", "bar", &dst)
	if err != errNoCodecs {
		t.Errorf("DecodeMulti: bad value for error, got: %v", err)
	}
}

func TestMissingKey(t *testing.T) {
	s1 := NewSecureCookie(nil, nil)

	var dst []byte
	err := s1.Decode("sid", "value", &dst)
	if err != errHashKeyNotSet {
		t.Fatalf("Expected %#v, got %#v", errHashKeyNotSet, err)
	}
	if err2, ok := err.(Error); !ok || !err2.IsUsage() {
		t.Errorf("Expected missing hash key to be IsUsage(); was %#v", err)
	}
}

// ----------------------------------------------------------------------------

type FooBar struct {
	Foo int
	Bar string
}

func TestCustomType(t *testing.T) {
	s1 := NewSecureCookie([]byte("12345"), []byte("1234567890123456"))
	// Type is not registered in gob. (!!!)
	src := &FooBar{42, "bar"}
	encoded, _ := s1.Encode("sid", src)

	dst := &FooBar{}
	_ = s1.Decode("sid", encoded, dst)
	if dst.Foo != 42 || dst.Bar != "bar" {
		t.Fatalf("Expected %#v, got %#v", src, dst)
	}
}
