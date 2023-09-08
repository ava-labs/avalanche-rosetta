package service

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"golang.org/x/crypto/sha3"
)

// constructContractCallDataGeneric constructs the data field of a transaction.
// The methodArgs can be already in ABI encoded format in case of a single string
// It can also be passed in as a slice of args, which requires further encoding.
func constructContractCallDataGeneric(methodSig string, methodArgs interface{}) ([]byte, error) {
	data, err := contractCallMethodID(methodSig)
	if err != nil {
		return nil, err
	}

	// switch on the type of the method args. method args can come in from json as either a string or list of strings
	switch methodArgs := methodArgs.(type) {
	// case 0: no method arguments, return the selector
	case nil:
		return data, nil

	// case 1: method args are pre-compiled ABI data. decode the hex and create the call data directly
	case string:
		methodArgs = strings.TrimPrefix(methodArgs, "0x")
		b, decErr := hex.DecodeString(methodArgs)
		if decErr != nil {
			return nil, fmt.Errorf("error decoding method args hex data: %w", decErr)
		}
		return append(data, b...), nil

	// case 2: method args are a list of interface{} which will be converted to string before encoding
	case []interface{}:
		var strList []string
		for i, genericVal := range methodArgs {
			strVal, isStrVal := genericVal.(string)
			if !isStrVal {
				return nil, fmt.Errorf("invalid method_args type at index %d: %T (must be a string)",
					i, genericVal,
				)
			}
			strList = append(strList, strVal)
		}
		return encodeMethodArgsStrings(data, methodSig, strList)

	// case 3: method args are encoded as a list of strings, which will be decoded
	case []string:
		return encodeMethodArgsStrings(data, methodSig, methodArgs)

	// case 4: there is no known way to decode the method args
	default:
		return nil, fmt.Errorf(
			"invalid method_args type, accepted values are []string and hex-encoded string."+
				" type received=%T value=%#v", methodArgs, methodArgs,
		)
	}
}

// encodeMethodArgsStrings constructs the data field of a transaction for a list of string args.
// It attempts to first convert the string arg to it's corresponding type in the method signature,
// and then performs abi encoding to the converted args list and construct the data.
func encodeMethodArgsStrings(methodID []byte, methodSig string, methodArgs []string) ([]byte, error) {
	arguments := abi.Arguments{}
	var argumentsData []interface{}

	var data []byte
	data = append(data, methodID...)

	const split = 2
	splitSigByLeadingParenthesis := strings.Split(methodSig, "(")
	if len(splitSigByLeadingParenthesis) < split {
		return data, nil
	}
	splitSigByTrailingParenthesis := strings.Split(splitSigByLeadingParenthesis[1], ")")
	if len(splitSigByTrailingParenthesis) < 1 {
		return data, nil
	}
	splitSigByComma := strings.Split(splitSigByTrailingParenthesis[0], ",")

	if len(splitSigByComma) != len(methodArgs) {
		return nil, errors.New("invalid method arguments")
	}

	for i, v := range splitSigByComma {
		typed, _ := abi.NewType(v, v, nil)
		argument := abi.Arguments{
			{
				Type: typed,
			},
		}

		arguments = append(arguments, argument...)
		var argData interface{}
		const base = 10
		switch {
		case v == "address":
			{
				argData = common.HexToAddress(methodArgs[i])
			}
		case v == "uint32":
			{
				u64, err := strconv.ParseUint(methodArgs[i], 10, 32)
				if err != nil {
					log.Fatal(err)
				}
				argData = uint32(u64)
			}
		case strings.HasPrefix(v, "uint") || strings.HasPrefix(v, "int"):
			{
				value := new(big.Int)
				value.SetString(methodArgs[i], base)
				argData = value
			}
		case v == "bytes32":
			{
				value := [32]byte{}
				bytes, err := hexutil.Decode(methodArgs[i])
				if err != nil {
					log.Fatal(err)
				}
				copy(value[:], bytes)
				argData = value
			}
		case strings.HasPrefix(v, "bytes"):
			{
				// No fixed size set as it would make it an "array" instead
				// of a "slice" when encoding. We want it to be a slice.
				value := []byte{}
				bytes, err := hexutil.Decode(methodArgs[i])
				if err != nil {
					log.Fatal(err)
				}
				copy(value[:], bytes) // nolint:gocritic
				argData = value
			}
		case strings.HasPrefix(v, "string"):
			{
				argData = methodArgs[i]
			}
		case strings.HasPrefix(v, "bool"):
			{
				value, err := strconv.ParseBool(methodArgs[i])
				if err != nil {
					log.Fatal(err)
				}
				argData = value
			}
		}
		argumentsData = append(argumentsData, argData)
	}

	abiEncodeData, err := arguments.PackValues(argumentsData)
	if err != nil {
		return nil, fmt.Errorf("failed to encode arguments: %w", err)
	}

	data = append(data, abiEncodeData...)
	return data, nil
}

// contractCallMethodID calculates the first 4 bytes of the method
// signature for function call on contract
func contractCallMethodID(methodSig string) ([]byte, error) {
	fnSignature := []byte(methodSig)
	hash := sha3.NewLegacyKeccak256()
	if _, err := hash.Write(fnSignature); err != nil {
		return nil, err
	}

	return hash.Sum(nil)[:4], nil
}
