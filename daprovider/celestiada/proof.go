// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package celestiada

import (
	"math/big"

	celestiaappproof "github.com/celestiaorg/celestia-app/v6/pkg/proof"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type Namespace struct {
	Version [1]byte
	Id      [28]byte
}

type NamespaceNode struct {
	Min    Namespace
	Max    Namespace
	Digest [32]byte
}

type BinaryMerkleProof struct {
	SideNodes [][32]byte
	Key       *big.Int
	NumLeaves *big.Int
}

type DataRootTuple struct {
	Height   *big.Int
	DataRoot [32]byte
}

type AttestationProof struct {
	TupleRootNonce *big.Int
	Tuple          DataRootTuple
	Proof          BinaryMerkleProof
}

type NamespaceMerkleMultiproof struct {
	BeginKey  *big.Int
	EndKey    *big.Int
	SideNodes []NamespaceNode
}

type SharesProof struct {
	Data             [][]byte
	ShareProofs      []NamespaceMerkleMultiproof
	Namespace        Namespace
	RowRoots         []NamespaceNode
	RowProofs        []BinaryMerkleProof
	AttestationProof AttestationProof
}

func toNamespaceNode(node []byte) NamespaceNode {
	var minID [28]byte
	copy(minID[:], node[1:29])
	var maxID [28]byte
	copy(maxID[:], node[30:58])
	var digest [32]byte
	copy(digest[:], node[58:])
	return NamespaceNode{
		Min: Namespace{
			Version: [1]byte{node[0]},
			Id:      minID,
		},
		Max: Namespace{
			Version: [1]byte{node[29]},
			Id:      maxID,
		},
		Digest: digest,
	}
}

func toRowProof(proof *celestiaappproof.Proof) BinaryMerkleProof {
	sideNodes := make([][32]byte, len(proof.Aunts))
	for i, sideNode := range proof.Aunts {
		copy(sideNodes[i][:], sideNode)
	}
	return BinaryMerkleProof{
		SideNodes: sideNodes,
		Key:       big.NewInt(proof.Index),
		NumLeaves: big.NewInt(proof.Total),
	}
}

func packSharesProof(blobstreamAddr common.Address, sharesProof SharesProof) ([]byte, error) {
	type namespaceABI struct {
		Version [1]byte
		Id      [28]byte
	}
	type namespaceNodeABI struct {
		Min    namespaceABI
		Max    namespaceABI
		Digest [32]byte
	}
	type binaryMerkleProofABI struct {
		SideNodes [][32]byte
		Key       *big.Int
		NumLeaves *big.Int
	}
	type namespaceMerkleMultiproofABI struct {
		BeginKey  *big.Int
		EndKey    *big.Int
		SideNodes []namespaceNodeABI
	}
	type dataRootTupleABI struct {
		Height   *big.Int
		DataRoot [32]byte
	}
	type attestationProofABI struct {
		TupleRootNonce *big.Int
		Tuple          dataRootTupleABI
		Proof          binaryMerkleProofABI
	}
	type sharesProofABI struct {
		Data             [][]byte
		ShareProofs      []namespaceMerkleMultiproofABI
		Namespace        namespaceABI
		RowRoots         []namespaceNodeABI
		RowProofs        []binaryMerkleProofABI
		AttestationProof attestationProofABI
	}

	args := abi.Arguments{
		{Name: "blobstream", Type: mustABIType("address", nil)},
		{
			Name: "sharesProof",
			Type: mustABIType("tuple", []abi.ArgumentMarshaling{
				{Name: "data", Type: "bytes[]"},
				{Name: "shareProofs", Type: "tuple[]", Components: []abi.ArgumentMarshaling{
					{Name: "beginKey", Type: "uint256"},
					{Name: "endKey", Type: "uint256"},
					{Name: "sideNodes", Type: "tuple[]", Components: []abi.ArgumentMarshaling{
						{Name: "min", Type: "tuple", Components: []abi.ArgumentMarshaling{{Name: "version", Type: "bytes1"}, {Name: "id", Type: "bytes28"}}},
						{Name: "max", Type: "tuple", Components: []abi.ArgumentMarshaling{{Name: "version", Type: "bytes1"}, {Name: "id", Type: "bytes28"}}},
						{Name: "digest", Type: "bytes32"},
					}},
				}},
				{Name: "namespace", Type: "tuple", Components: []abi.ArgumentMarshaling{{Name: "version", Type: "bytes1"}, {Name: "id", Type: "bytes28"}}},
				{Name: "rowRoots", Type: "tuple[]", Components: []abi.ArgumentMarshaling{
					{Name: "min", Type: "tuple", Components: []abi.ArgumentMarshaling{{Name: "version", Type: "bytes1"}, {Name: "id", Type: "bytes28"}}},
					{Name: "max", Type: "tuple", Components: []abi.ArgumentMarshaling{{Name: "version", Type: "bytes1"}, {Name: "id", Type: "bytes28"}}},
					{Name: "digest", Type: "bytes32"},
				}},
				{Name: "rowProofs", Type: "tuple[]", Components: []abi.ArgumentMarshaling{
					{Name: "sideNodes", Type: "bytes32[]"},
					{Name: "key", Type: "uint256"},
					{Name: "numLeaves", Type: "uint256"},
				}},
				{Name: "attestationProof", Type: "tuple", Components: []abi.ArgumentMarshaling{
					{Name: "tupleRootNonce", Type: "uint256"},
					{Name: "tuple", Type: "tuple", Components: []abi.ArgumentMarshaling{{Name: "height", Type: "uint256"}, {Name: "dataRoot", Type: "bytes32"}}},
					{Name: "proof", Type: "tuple", Components: []abi.ArgumentMarshaling{
						{Name: "sideNodes", Type: "bytes32[]"},
						{Name: "key", Type: "uint256"},
						{Name: "numLeaves", Type: "uint256"},
					}},
				}},
			}),
		},
	}

	shareProofs := make([]namespaceMerkleMultiproofABI, 0, len(sharesProof.ShareProofs))
	for _, shareProof := range sharesProof.ShareProofs {
		sideNodes := make([]namespaceNodeABI, 0, len(shareProof.SideNodes))
		for _, sideNode := range shareProof.SideNodes {
			sideNodes = append(sideNodes, namespaceNodeABI{
				Min:    namespaceABI(sideNode.Min),
				Max:    namespaceABI(sideNode.Max),
				Digest: sideNode.Digest,
			})
		}
		shareProofs = append(shareProofs, namespaceMerkleMultiproofABI{
			BeginKey:  shareProof.BeginKey,
			EndKey:    shareProof.EndKey,
			SideNodes: sideNodes,
		})
	}

	rowRoots := make([]namespaceNodeABI, 0, len(sharesProof.RowRoots))
	for _, rowRoot := range sharesProof.RowRoots {
		rowRoots = append(rowRoots, namespaceNodeABI{
			Min:    namespaceABI(rowRoot.Min),
			Max:    namespaceABI(rowRoot.Max),
			Digest: rowRoot.Digest,
		})
	}

	rowProofs := make([]binaryMerkleProofABI, 0, len(sharesProof.RowProofs))
	for _, rowProof := range sharesProof.RowProofs {
		rowProofs = append(rowProofs, binaryMerkleProofABI{
			SideNodes: rowProof.SideNodes,
			Key:       rowProof.Key,
			NumLeaves: rowProof.NumLeaves,
		})
	}

	return args.Pack(
		blobstreamAddr,
		sharesProofABI{
			Data:        sharesProof.Data,
			ShareProofs: shareProofs,
			Namespace: namespaceABI{
				Version: sharesProof.Namespace.Version,
				Id:      sharesProof.Namespace.Id,
			},
			RowRoots:  rowRoots,
			RowProofs: rowProofs,
			AttestationProof: attestationProofABI{
				TupleRootNonce: sharesProof.AttestationProof.TupleRootNonce,
				Tuple: dataRootTupleABI{
					Height:   sharesProof.AttestationProof.Tuple.Height,
					DataRoot: sharesProof.AttestationProof.Tuple.DataRoot,
				},
				Proof: binaryMerkleProofABI{
					SideNodes: sharesProof.AttestationProof.Proof.SideNodes,
					Key:       sharesProof.AttestationProof.Proof.Key,
					NumLeaves: sharesProof.AttestationProof.Proof.NumLeaves,
				},
			},
		},
	)
}

func packValidityProof(attestationProof AttestationProof) ([]byte, error) {
	type binaryMerkleProofABI struct {
		SideNodes [][32]byte
		Key       *big.Int
		NumLeaves *big.Int
	}
	type dataRootTupleABI struct {
		Height   *big.Int
		DataRoot [32]byte
	}
	type attestationProofABI struct {
		TupleRootNonce *big.Int
		Tuple          dataRootTupleABI
		Proof          binaryMerkleProofABI
	}

	args := abi.Arguments{
		{
			Name: "attestationProof",
			Type: mustABIType("tuple", []abi.ArgumentMarshaling{
				{Name: "tupleRootNonce", Type: "uint256"},
				{Name: "tuple", Type: "tuple", Components: []abi.ArgumentMarshaling{
					{Name: "height", Type: "uint256"},
					{Name: "dataRoot", Type: "bytes32"},
				}},
				{Name: "proof", Type: "tuple", Components: []abi.ArgumentMarshaling{
					{Name: "sideNodes", Type: "bytes32[]"},
					{Name: "key", Type: "uint256"},
					{Name: "numLeaves", Type: "uint256"},
				}},
			}),
		},
	}

	return args.Pack(attestationProofABI{
		TupleRootNonce: attestationProof.TupleRootNonce,
		Tuple: dataRootTupleABI{
			Height:   attestationProof.Tuple.Height,
			DataRoot: attestationProof.Tuple.DataRoot,
		},
		Proof: binaryMerkleProofABI{
			SideNodes: attestationProof.Proof.SideNodes,
			Key:       attestationProof.Proof.Key,
			NumLeaves: attestationProof.Proof.NumLeaves,
		},
	})
}

func mustABIType(name string, components []abi.ArgumentMarshaling) abi.Type {
	typ, err := abi.NewType(name, "", components)
	if err != nil {
		panic("invalid ABI type: " + name)
	}
	return typ
}
