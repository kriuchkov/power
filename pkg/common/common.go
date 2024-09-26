package common

import (
	"fmt"
	"strconv"
	"strings"
)

func ConvetVerfyMessageToBytes(primaryHash []byte, byteIndex int, byteValue byte) []byte {
	return []byte(fmt.Sprintf("%s|%d|%d", primaryHash, byteIndex, byteValue))
}

//nolint:nonamedreturns // it's a helper function
func SplitMessage(body []byte) (hash []byte, byteIndex int, byteValue byte) {
	split := strings.Split(string(body), "|")
	if len(split) != 3 {
		return nil, 0, 0
	}

	hash = []byte(split[0])
	byteIndex, _ = strconv.Atoi(split[1])
	byteValueInt, _ := strconv.Atoi(split[2])
	byteValue = byte(byteValueInt)
	return hash, byteIndex, byteValue
}

func GetNonceFromMessage(data []byte) int {
	clientNonce, _ := strconv.Atoi(string(data))
	return clientNonce
}
