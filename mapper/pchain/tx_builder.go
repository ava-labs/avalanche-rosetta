package pchain

import "github.com/ava-labs/avalanche-rosetta/mapper"

func ParseOpMetadata(metadata map[string]interface{}) (*OperationMetadata, error) {
	var operationMetadata OperationMetadata
	if err := mapper.UnmarshalJSONMap(metadata, &operationMetadata); err != nil {
		return nil, err
	}

	// set threshold default to 1
	if operationMetadata.Threshold == 0 {
		operationMetadata.Threshold = 1
	}

	// set sig indices to a single signer if not provided
	if operationMetadata.SigIndices == nil {
		operationMetadata.SigIndices = []uint32{0}
	}

	return &operationMetadata, nil
}
