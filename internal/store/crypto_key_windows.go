//go:build windows

package store

import (
	"fmt"
	"syscall"
	"unsafe"
)

const cryptProtectUIForbidden = 0x1

var (
	crypt32                = syscall.NewLazyDLL("crypt32.dll")
	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	procCryptProtectData   = crypt32.NewProc("CryptProtectData")
	procCryptUnprotectData = crypt32.NewProc("CryptUnprotectData")
	procLocalFree          = kernel32.NewProc("LocalFree")
)

type dataBlob struct {
	cbData uint32
	pbData *byte
}

func protectSecretKey(key []byte) ([]byte, bool, error) {
	if len(key) == 0 {
		return nil, false, fmt.Errorf("secret key is empty")
	}
	in := dataBlob{cbData: uint32(len(key)), pbData: &key[0]}
	var out dataBlob
	ok, _, err := procCryptProtectData.Call(
		uintptr(unsafe.Pointer(&in)),
		0,
		0,
		0,
		0,
		uintptr(cryptProtectUIForbidden),
		uintptr(unsafe.Pointer(&out)),
	)
	if ok == 0 {
		return nil, false, err
	}
	defer procLocalFree.Call(uintptr(unsafe.Pointer(out.pbData)))
	return blobBytes(out), true, nil
}

func unprotectSecretKey(protected []byte) ([]byte, error) {
	if len(protected) == 0 {
		return nil, fmt.Errorf("protected secret key is empty")
	}
	in := dataBlob{cbData: uint32(len(protected)), pbData: &protected[0]}
	var out dataBlob
	ok, _, err := procCryptUnprotectData.Call(
		uintptr(unsafe.Pointer(&in)),
		0,
		0,
		0,
		0,
		uintptr(cryptProtectUIForbidden),
		uintptr(unsafe.Pointer(&out)),
	)
	if ok == 0 {
		return nil, err
	}
	defer procLocalFree.Call(uintptr(unsafe.Pointer(out.pbData)))
	return blobBytes(out), nil
}

func blobBytes(blob dataBlob) []byte {
	if blob.cbData == 0 || blob.pbData == nil {
		return nil
	}
	src := unsafe.Slice(blob.pbData, blob.cbData)
	out := make([]byte, len(src))
	copy(out, src)
	return out
}
