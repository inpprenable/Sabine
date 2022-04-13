package Blockchain

import (
	"crypto/ed25519"
	"time"
)

type ValidatorInterf interface {
	ValidatorGetterInterf
	GenerateAddresses(numberOfValidators int)
	IsSizeValid(newSize int) bool
	IsActiveValidator(key ed25519.PublicKey) bool
	IsValidator(key ed25519.PublicKey) bool
	SetNumberOfNode(int) bool
	Close()
	GetIndexOfValidator(key ed25519.PublicKey) int
	GetValidatorOfIndex(int) ed25519.PublicKey
}

type ValidatorGetterInterf interface {
	GetNumberOfValidator() int
	IsOlderThan(timestamp time.Duration) bool
}

type MetricGetterInterf interface {
	ValidatorGetterInterf
	GetIncQueueSize() int
}
