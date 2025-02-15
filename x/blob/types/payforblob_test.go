package types_test

import (
	"bytes"
	"testing"

	sdkerrors "cosmossdk.io/errors"
	"github.com/celestiaorg/celestia-app/pkg/appconsts"
	"github.com/celestiaorg/celestia-app/pkg/blob"
	appns "github.com/celestiaorg/celestia-app/pkg/namespace"
	shares "github.com/celestiaorg/celestia-app/pkg/shares"
	"github.com/celestiaorg/celestia-app/test/util/testfactory"
	"github.com/celestiaorg/celestia-app/test/util/testnode"
	"github.com/celestiaorg/celestia-app/x/blob/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tmrand "github.com/tendermint/tendermint/libs/rand"
)

func Test_MerkleMountainRangeHeights(t *testing.T) {
	type test struct {
		totalSize  uint64
		squareSize uint64
		expected   []uint64
	}
	tests := []test{
		{
			totalSize:  11,
			squareSize: 4,
			expected:   []uint64{4, 4, 2, 1},
		},
		{
			totalSize:  2,
			squareSize: 64,
			expected:   []uint64{2},
		},
		{
			totalSize:  64,
			squareSize: 8,
			expected:   []uint64{8, 8, 8, 8, 8, 8, 8, 8},
		},
		// Height
		// 3              x                               x
		//              /    \                         /    \
		//             /      \                       /      \
		//            /        \                     /        \
		//           /          \                   /          \
		// 2        x            x                 x            x
		//        /   \        /   \             /   \        /   \
		// 1     x     x      x     x           x     x      x     x         x
		//      / \   / \    / \   / \         / \   / \    / \   / \      /   \
		// 0   0   1 2   3  4   5 6   7       8   9 10  11 12 13 14  15   16   17    18
		{
			totalSize:  19,
			squareSize: 8,
			expected:   []uint64{8, 8, 2, 1},
		},
	}
	for _, tt := range tests {
		res, err := types.MerkleMountainRangeSizes(tt.totalSize, tt.squareSize)
		require.NoError(t, err)
		assert.Equal(t, tt.expected, res)
	}
}

// TestCreateCommitment will fail if a change is made to share encoding or how
// the commitment is calculated. If this is the case, the expected commitment
// bytes will need to be updated.
func TestCreateCommitment(t *testing.T) {
	ns1 := appns.MustNewV0(bytes.Repeat([]byte{0x1}, appns.NamespaceVersionZeroIDSize))

	type test struct {
		name         string
		namespace    appns.Namespace
		blob         []byte
		expected     []byte
		expectErr    bool
		shareVersion uint8
	}
	tests := []test{
		{
			name:         "blob of 3 shares succeeds",
			namespace:    ns1,
			blob:         bytes.Repeat([]byte{0xFF}, 3*appconsts.ShareSize),
			expected:     []byte{0x3b, 0x9e, 0x78, 0xb6, 0x64, 0x8e, 0xc1, 0xa2, 0x41, 0x92, 0x5b, 0x31, 0xda, 0x2e, 0xcb, 0x50, 0xbf, 0xc6, 0xf4, 0xad, 0x55, 0x2d, 0x32, 0x79, 0x92, 0x8c, 0xa1, 0x3e, 0xbe, 0xba, 0x8c, 0x2b},
			shareVersion: appconsts.ShareVersionZero,
		},
		{
			name:         "blob with unsupported share version should return error",
			namespace:    ns1,
			blob:         bytes.Repeat([]byte{0xFF}, 12*appconsts.ShareSize),
			expectErr:    true,
			shareVersion: uint8(1), // unsupported share version
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blob := &blob.Blob{
				NamespaceId:      tt.namespace.ID,
				Data:             tt.blob,
				ShareVersion:     uint32(tt.shareVersion),
				NamespaceVersion: uint32(tt.namespace.Version),
			}
			res, err := types.CreateCommitment(blob)
			if tt.expectErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, res)
		})
	}
}

func TestMsgTypeURLParity(t *testing.T) {
	require.Equal(t, sdk.MsgTypeURL(&types.MsgPayForBlobs{}), types.URLMsgPayForBlobs)
}

func TestValidateBasic(t *testing.T) {
	type test struct {
		name    string
		msg     *types.MsgPayForBlobs
		wantErr *sdkerrors.Error
	}

	validMsg := validMsgPayForBlobs(t)

	// MsgPayForBlobs that uses parity shares namespace
	paritySharesMsg := validMsgPayForBlobs(t)
	paritySharesMsg.Namespaces[0] = appns.ParitySharesNamespace.Bytes()

	// MsgPayForBlobs that uses tail padding namespace
	tailPaddingMsg := validMsgPayForBlobs(t)
	tailPaddingMsg.Namespaces[0] = appns.TailPaddingNamespace.Bytes()

	// MsgPayForBlobs that uses transaction namespace
	txNamespaceMsg := validMsgPayForBlobs(t)
	txNamespaceMsg.Namespaces[0] = appns.TxNamespace.Bytes()

	// MsgPayForBlobs that uses intermediateStateRoots namespace
	intermediateStateRootsNamespaceMsg := validMsgPayForBlobs(t)
	intermediateStateRootsNamespaceMsg.Namespaces[0] = appns.IntermediateStateRootsNamespace.Bytes()

	// MsgPayForBlobs that uses the max primary reserved namespace
	maxReservedNamespaceMsg := validMsgPayForBlobs(t)
	maxReservedNamespaceMsg.Namespaces[0] = appns.MaxPrimaryReservedNamespace.Bytes()

	// MsgPayForBlobs that has an empty share commitment
	emptyShareCommitment := validMsgPayForBlobs(t)
	emptyShareCommitment.ShareCommitments[0] = []byte{}

	// MsgPayForBlobs that has an invalid share commitment size
	invalidShareCommitmentSize := validMsgPayForBlobs(t)
	invalidShareCommitmentSize.ShareCommitments[0] = bytes.Repeat([]byte{0x1}, 31)

	// MsgPayForBlobs that has no namespaces
	noNamespaces := validMsgPayForBlobs(t)
	noNamespaces.Namespaces = [][]byte{}

	// MsgPayForBlobs that has no share versions
	noShareVersions := validMsgPayForBlobs(t)
	noShareVersions.ShareVersions = []uint32{}

	// MsgPayForBlobs that has no blob sizes
	noBlobSizes := validMsgPayForBlobs(t)
	noBlobSizes.BlobSizes = []uint32{}

	// MsgPayForBlobs that has no share commitments
	noShareCommitments := validMsgPayForBlobs(t)
	noShareCommitments.ShareCommitments = [][]byte{}

	tests := []test{
		{
			name:    "valid msg",
			msg:     validMsg,
			wantErr: nil,
		},
		{
			name:    "parity shares namespace",
			msg:     paritySharesMsg,
			wantErr: types.ErrReservedNamespace,
		},
		{
			name:    "tail padding namespace",
			msg:     tailPaddingMsg,
			wantErr: types.ErrReservedNamespace,
		},
		{
			name:    "tx namespace",
			msg:     txNamespaceMsg,
			wantErr: types.ErrReservedNamespace,
		},
		{
			name:    "intermediate state root namespace",
			msg:     intermediateStateRootsNamespaceMsg,
			wantErr: types.ErrReservedNamespace,
		},
		{
			name:    "max reserved namespace",
			msg:     maxReservedNamespaceMsg,
			wantErr: types.ErrReservedNamespace,
		},
		{
			name:    "empty share commitment",
			msg:     emptyShareCommitment,
			wantErr: types.ErrInvalidShareCommitment,
		},
		{
			name:    "incorrect hash size share commitment",
			msg:     invalidShareCommitmentSize,
			wantErr: types.ErrInvalidShareCommitment,
		},
		{
			name:    "no namespace ids",
			msg:     noNamespaces,
			wantErr: types.ErrNoNamespaces,
		},
		{
			name:    "no share versions",
			msg:     noShareVersions,
			wantErr: types.ErrNoShareVersions,
		},
		{
			name:    "no blob sizes",
			msg:     noBlobSizes,
			wantErr: types.ErrNoBlobSizes,
		},
		{
			name:    "no share commitments",
			msg:     noShareCommitments,
			wantErr: types.ErrNoShareCommitments,
		},
		{
			name:    "invalid namespace version",
			msg:     invalidNamespaceVersionMsgPayForBlobs(t),
			wantErr: types.ErrInvalidNamespaceVersion,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr.Error())
				space, code, log := sdkerrors.ABCIInfo(err, false)
				assert.Equal(t, tt.wantErr.Codespace(), space)
				assert.Equal(t, tt.wantErr.ABCICode(), code)
				t.Log(log)
			}
		})
	}
}

// totalBlobSize subtracts the delimiter size from the desired total size. this
// is useful for testing for blobs that occupy exactly so many shares.
func totalBlobSize(size int) int {
	return size - shares.DelimLen(uint64(size))
}

func validMsgPayForBlobs(t *testing.T) *types.MsgPayForBlobs {
	signer, err := testnode.NewOfflineSigner()
	require.NoError(t, err)
	ns1 := append(appns.NamespaceVersionZeroPrefix, bytes.Repeat([]byte{0x01}, appns.NamespaceVersionZeroIDSize)...)
	data := bytes.Repeat([]byte{2}, totalBlobSize(appconsts.ContinuationSparseShareContentSize*12))

	pblob := &blob.Blob{
		Data:             data,
		NamespaceId:      ns1,
		NamespaceVersion: uint32(appns.NamespaceVersionZero),
		ShareVersion:     uint32(appconsts.ShareVersionZero),
	}

	addr := signer.Address()
	pfb, err := types.NewMsgPayForBlobs(addr.String(), pblob)
	assert.NoError(t, err)

	return pfb
}

func invalidNamespaceVersionMsgPayForBlobs(t *testing.T) *types.MsgPayForBlobs {
	signer, err := testnode.NewOfflineSigner()
	require.NoError(t, err)
	ns1 := append(appns.NamespaceVersionZeroPrefix, bytes.Repeat([]byte{0x01}, appns.NamespaceVersionZeroIDSize)...)
	data := bytes.Repeat([]byte{2}, totalBlobSize(appconsts.ContinuationSparseShareContentSize*12))

	pblob := &blob.Blob{
		Data:             data,
		NamespaceId:      ns1,
		NamespaceVersion: uint32(255),
		ShareVersion:     uint32(appconsts.ShareVersionZero),
	}

	blobs := []*blob.Blob{pblob}

	commitments, err := types.CreateCommitments(blobs)
	require.NoError(t, err)

	namespaceVersions, namespaceIds, sizes, shareVersions := types.ExtractBlobComponents(blobs)
	namespaces := []appns.Namespace{}
	for i := range namespaceVersions {
		namespace, err := appns.New(uint8(namespaceVersions[i]), namespaceIds[i])
		require.NoError(t, err)
		namespaces = append(namespaces, namespace)
	}

	namespacesBytes := make([][]byte, len(namespaces))
	for idx, namespace := range namespaces {
		namespacesBytes[idx] = namespace.Bytes()
	}

	addr := signer.Address()
	pfb := &types.MsgPayForBlobs{
		Signer:           addr.String(),
		Namespaces:       namespacesBytes,
		ShareCommitments: commitments,
		BlobSizes:        sizes,
		ShareVersions:    shareVersions,
	}

	return pfb
}

func TestNewMsgPayForBlobs(t *testing.T) {
	type testCase struct {
		name        string
		signer      string
		blobs       []*blob.Blob
		expectedErr bool
	}
	ns1 := appns.MustNewV0(bytes.Repeat([]byte{1}, appns.NamespaceVersionZeroIDSize))
	ns2 := appns.MustNewV0(bytes.Repeat([]byte{2}, appns.NamespaceVersionZeroIDSize))

	testCases := []testCase{
		{
			name:   "valid msg PFB with small blob",
			signer: testfactory.TestAccAddr,
			blobs: []*blob.Blob{
				{
					NamespaceVersion: uint32(ns1.Version),
					NamespaceId:      ns1.ID,
					Data:             []byte{1},
					ShareVersion:     uint32(appconsts.ShareVersionZero),
				},
			},
			expectedErr: false,
		},
		{
			name:   "valid msg PFB with large blob",
			signer: testfactory.TestAccAddr,
			blobs: []*blob.Blob{
				{
					NamespaceVersion: uint32(ns1.Version),
					NamespaceId:      ns1.ID,
					Data:             tmrand.Bytes(1000000),
					ShareVersion:     uint32(appconsts.ShareVersionZero),
				},
			},
			expectedErr: false,
		},
		{
			name:   "valid msg PFB with two blobs",
			signer: testfactory.TestAccAddr,
			blobs: []*blob.Blob{
				{
					NamespaceVersion: uint32(ns1.Version),
					NamespaceId:      ns1.ID,
					Data:             []byte{1},
					ShareVersion:     uint32(appconsts.ShareVersionZero),
				},
				{
					NamespaceVersion: uint32(ns2.Version),
					NamespaceId:      ns2.ID,
					Data:             []byte{2},
					ShareVersion:     uint32(appconsts.ShareVersionZero),
				},
			},
		},
		{
			name:   "unsupported share version returns an error",
			signer: testfactory.TestAccAddr,
			blobs: []*blob.Blob{
				{
					NamespaceVersion: uint32(ns1.Version),
					NamespaceId:      ns1.ID,
					Data:             tmrand.Bytes(1000000),
					ShareVersion:     uint32(10), // unsupported share version
				},
			},
			expectedErr: true,
		},
		{
			name:   "msg PFB with tx namespace returns an error",
			signer: testfactory.TestAccAddr,
			blobs: []*blob.Blob{
				{
					NamespaceVersion: uint32(appns.TxNamespace.Version),
					NamespaceId:      appns.TxNamespace.ID,
					Data:             tmrand.Bytes(1000000),
					ShareVersion:     uint32(appconsts.ShareVersionZero),
				},
			},
			expectedErr: true,
		},
		{
			name:   "msg PFB with invalid signer returns an error",
			signer: testfactory.TestAccAddr[:10],
			blobs: []*blob.Blob{
				{
					NamespaceVersion: uint32(ns1.Version),
					NamespaceId:      ns1.ID,
					Data:             []byte{1},
					ShareVersion:     uint32(appconsts.ShareVersionZero),
				},
			},
			expectedErr: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			msgPFB, err := types.NewMsgPayForBlobs(tc.signer, tc.blobs...)
			if tc.expectedErr {
				assert.Error(t, err)
				return
			}

			for i, blob := range tc.blobs {
				assert.Equal(t, uint32(len(blob.Data)), msgPFB.BlobSizes[i])
				ns, err := appns.From(msgPFB.Namespaces[i])
				assert.NoError(t, err)
				assert.Equal(t, ns.ID, blob.NamespaceId)
				assert.Equal(t, uint32(ns.Version), blob.NamespaceVersion)

				expectedCommitment, err := types.CreateCommitment(blob)
				require.NoError(t, err)
				assert.Equal(t, expectedCommitment, msgPFB.ShareCommitments[i])
			}
		})
	}
}

func TestValidateBlobs(t *testing.T) {
	type test struct {
		name        string
		blob        *blob.Blob
		expectError bool
	}

	tests := []test{
		{
			name: "valid blob",
			blob: &blob.Blob{
				Data:             []byte{1},
				NamespaceId:      appns.RandomBlobNamespace().ID,
				ShareVersion:     uint32(appconsts.DefaultShareVersion),
				NamespaceVersion: uint32(appns.NamespaceVersionZero),
			},
			expectError: false,
		},
		{
			name: "invalid share version",
			blob: &blob.Blob{
				Data:             []byte{1},
				NamespaceId:      appns.RandomBlobNamespace().ID,
				ShareVersion:     uint32(10000),
				NamespaceVersion: uint32(appns.NamespaceVersionZero),
			},
			expectError: true,
		},
		{
			name: "empty blob",
			blob: &blob.Blob{
				Data:             []byte{},
				NamespaceId:      appns.RandomBlobNamespace().ID,
				ShareVersion:     uint32(appconsts.DefaultShareVersion),
				NamespaceVersion: uint32(appns.NamespaceVersionZero),
			},
			expectError: true,
		},
		{
			name: "invalid namespace",
			blob: &blob.Blob{
				Data:             []byte{1},
				NamespaceId:      appns.TxNamespace.ID,
				ShareVersion:     uint32(appconsts.DefaultShareVersion),
				NamespaceVersion: uint32(appns.NamespaceVersionZero),
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		err := types.ValidateBlobs(tt.blob)
		if tt.expectError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	}
}
