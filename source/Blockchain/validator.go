package Blockchain

import (
	"bytes"
	"crypto/ed25519"
	"fmt"
	"time"
)

type Validators struct {
	refList    []ed25519.PublicKey
	List       []ed25519.PublicKey
	lastChange int64
}

func (valid *Validators) IsSizeValid(newSize int) bool {
	return newSize >= 4 && newSize <= len(valid.refList)
}

func (valid Validators) GetNumberOfValidator() int {
	return len(valid.List)
}

func (valid *Validators) SetNumberOfNode(newSize int) bool {
	if valid.IsSizeValid(newSize) {
		if len(valid.List) != newSize {
			valid.List = valid.refList[:min(newSize, len(valid.refList))]
		}
		valid.lastChange = time.Now().UnixNano()
		return true
	}
	return false
}

// IsOlderThan return true if more than timestamp nanosecond have been passed after a new block
func (valid Validators) IsOlderThan(timestamp time.Duration) bool {
	return time.Now().UnixNano()-valid.lastChange > timestamp.Nanoseconds()
}

func (valid *Validators) Close() {
	panic("implement me")
}

func (valid Validators) GetIndexOfValidator(key ed25519.PublicKey) int {
	for i, a := range valid.refList {
		if bytes.Equal(key, a) {
			return i
		}
	}
	return -1
}

func (valid Validators) GetValidatorOfIndex(i int) ed25519.PublicKey {
	return valid.refList[i]
}

func (v *Validators) GenerateAddresses(numberOfValidators int) {
	for input := 0; input < numberOfValidators; input++ {
		v.refList = append(v.refList, NewWallet(fmt.Sprintf("NODE%d", input)).PublicKey())
	}
	v.SetNumberOfNode(numberOfValidators)
}

// IsActiveValidator Return if the key is an Active validator
func (v Validators) IsActiveValidator(key ed25519.PublicKey) bool {
	for _, a := range v.List {
		if bytes.Equal(key, a) {
			return true
		}
	}
	return false
}

// IsValidator Return if the key is an Active or Inactive validator
func (v Validators) IsValidator(key ed25519.PublicKey) bool {
	for _, a := range v.refList {
		if bytes.Equal(key, a) {
			return true
		}
	}
	return false
}

func (v Validators) GetValidatorIndex(key ed25519.PublicKey) int {
	for i, a := range v.refList {
		if bytes.Equal(key, a) {
			return i
		}
	}
	return -1
}
