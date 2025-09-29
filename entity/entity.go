package entity

import (
	"crypto/sha256"
	"errors"
	"fmt"
)

const MaxPackLen = 1 << 16

type Request struct {
	Number    int64
	CreatedAt int64
	Payload   []byte
}

type Response struct {
	Number     int64
	CreatedAt  int64
	ReceivedAt int64
}

func NewPack(data []byte) ([]byte, error) {
	checksum := sha256.Sum256(data)
	pack := append(checksum[:], data...)
	if l := len(pack); l > MaxPackLen {
		return nil, fmt.Errorf("pack is too long: %v (max %v)", l, MaxPackLen)
	}
	return pack, nil
}

func Unpack(data []byte) ([]byte, error) {
	if l := len(data); l < 32 {
		return nil, fmt.Errorf("data is too short: %v (min 32)", l)
	}
	payload := data[32:]
	if [32]byte(data[:32]) != sha256.Sum256(payload) {
		return nil, errors.New("checksum mismatch: corrupted pack")
	}
	return payload, nil
}
