// Adapted from the BLBX interpreter to operate on compiled graph components.
package graphvm

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"unicode/utf8"
)

const binaryLimit = 1 << 20

func addSecurity(add nativeAdder) {
	for _, name := range []string{"random_bytes", "random_hex"} {
		n := name
		add(n, []Kind{IntegerKind}, func(a []Value) (Value, error) {
			count := a[0].Integer
			if count < 0 || count > binaryLimit {
				return Null(), fmt.Errorf("byte count must be between 0 and 1048576")
			}
			b := make([]byte, int(count))
			if _, err := rand.Read(b); err != nil {
				return Null(), err
			}
			if n == "random_hex" {
				return String(hex.EncodeToString(b)), nil
			}
			return byteArray(b), nil
		})
	}
	add("random_int", []Kind{IntegerKind, IntegerKind}, func(a []Value) (Value, error) {
		min, max := a[0].Integer, a[1].Integer
		if min >= max {
			return Null(), fmt.Errorf("minimum must be less than exclusive maximum")
		}
		width := new(big.Int).Sub(big.NewInt(max), big.NewInt(min))
		n, err := rand.Int(rand.Reader, width)
		if err != nil {
			return Null(), err
		}
		return Integer(n.Add(n, big.NewInt(min)).Int64()), nil
	})
	for _, name := range []string{"md5", "sha256", "sha512", "hex_encode", "base64_encode"} {
		n := name
		add(n, []Kind{""}, func(a []Value) (Value, error) {
			data, err := valueBytes(a[0])
			if err != nil {
				return Null(), err
			}
			switch n {
			case "md5":
				digest := md5.Sum(data)
				return String(hex.EncodeToString(digest[:])), nil
			case "sha256":
				digest := sha256.Sum256(data)
				return String(hex.EncodeToString(digest[:])), nil
			case "sha512":
				digest := sha512.Sum512(data)
				return String(hex.EncodeToString(digest[:])), nil
			case "hex_encode":
				return String(hex.EncodeToString(data)), nil
			default:
				return String(base64.StdEncoding.EncodeToString(data)), nil
			}
		})
	}
	for _, name := range []string{"hex_decode", "base64_decode"} {
		n := name
		add(n, []Kind{StringKind}, func(a []Value) (Value, error) {
			if len(a[0].String) > 2*binaryLimit {
				return Null(), fmt.Errorf("encoded input exceeds limit")
			}
			var b []byte
			var err error
			if n == "hex_decode" {
				b, err = hex.DecodeString(a[0].String)
			} else {
				b, err = base64.StdEncoding.Strict().DecodeString(a[0].String)
			}
			if err != nil {
				return Null(), err
			}
			if len(b) > binaryLimit {
				return Null(), fmt.Errorf("decoded bytes exceed 1 MiB")
			}
			return byteArray(b), nil
		})
	}
	add("utf8_encode", []Kind{StringKind}, func(a []Value) (Value, error) {
		if !utf8.ValidString(a[0].String) {
			return Null(), fmt.Errorf("invalid UTF-8")
		}
		if len(a[0].String) > binaryLimit {
			return Null(), fmt.Errorf("encoded bytes exceed 1 MiB")
		}
		return byteArray([]byte(a[0].String)), nil
	})
	add("utf8_decode", []Kind{ArrayKind}, func(a []Value) (Value, error) {
		b, e := valueBytes(a[0])
		if e != nil {
			return Null(), e
		}
		if !utf8.Valid(b) {
			return Null(), fmt.Errorf("invalid UTF-8")
		}
		return String(string(b)), nil
	})
	add("hmac_sha256", []Kind{"", ""}, func(a []Value) (Value, error) {
		key, e := valueBytes(a[0])
		if e != nil {
			return Null(), e
		}
		data, e := valueBytes(a[1])
		if e != nil {
			return Null(), e
		}
		mac := hmac.New(sha256.New, key)
		mac.Write(data)
		return String(hex.EncodeToString(mac.Sum(nil))), nil
	})
	add("verify_hmac_sha256", []Kind{"", "", StringKind}, func(a []Value) (Value, error) {
		key, e := valueBytes(a[0])
		if e != nil {
			return Null(), e
		}
		data, e := valueBytes(a[1])
		if e != nil {
			return Null(), e
		}
		if len(a[2].String) != 64 {
			return Null(), fmt.Errorf("expected HMAC must be 64 hexadecimal characters")
		}
		expected, e := hex.DecodeString(a[2].String)
		if e != nil || len(expected) != sha256.Size {
			return Null(), fmt.Errorf("expected HMAC must be 64 hexadecimal characters")
		}
		mac := hmac.New(sha256.New, key)
		mac.Write(data)
		return Boolean(hmac.Equal(mac.Sum(nil), expected)), nil
	})
	add("constant_time_equal", []Kind{"", ""}, func(a []Value) (Value, error) {
		x, e := valueBytes(a[0])
		if e != nil {
			return Null(), e
		}
		y, e := valueBytes(a[1])
		if e != nil {
			return Null(), e
		}
		return Boolean(hmac.Equal(x, y)), nil
	})
}
