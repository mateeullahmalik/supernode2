package supernodeservice

import (
	"testing"

	"github.com/LumeraProtocol/supernode/gen/supernode/action/cascade"
	"github.com/LumeraProtocol/supernode/sdk/event"
	"github.com/stretchr/testify/require"
)

func TestTranslateSupernodeEvent(t *testing.T) {
	tests := []struct {
		name     string
		input    cascade.SupernodeEventType
		expected event.EventType
	}{
		{"ACTION_RETRIEVED", cascade.SupernodeEventType_ACTION_RETRIEVED, event.TaskProgressActionRetrievedBySupernode},
		{"ACTION_FEE_VERIFIED", cascade.SupernodeEventType_ACTION_FEE_VERIFIED, event.TaskProgressActionFeeValidated},
		{"TOP_SUPERNODE_CHECK_PASSED", cascade.SupernodeEventType_TOP_SUPERNODE_CHECK_PASSED, event.TaskProgressTopSupernodeCheckValidated},
		{"METADATA_DECODED", cascade.SupernodeEventType_METADATA_DECODED, event.TaskProgressCascadeMetadataDecoded},
		{"DATA_HASH_VERIFIED", cascade.SupernodeEventType_DATA_HASH_VERIFIED, event.TaskProgressDataHashVerified},
		{"INPUT_ENCODED", cascade.SupernodeEventType_INPUT_ENCODED, event.TaskProgressInputDataEncoded},
		{"SIGNATURE_VERIFIED", cascade.SupernodeEventType_SIGNATURE_VERIFIED, event.TaskProgressSignatureVerified},
		{"RQID_GENERATED", cascade.SupernodeEventType_RQID_GENERATED, event.TaskProgressRQIDFilesGenerated},
		{"RQID_VERIFIED", cascade.SupernodeEventType_RQID_VERIFIED, event.TaskProgressRQIDsVerified},
		{"ARTEFACTS_STORED", cascade.SupernodeEventType_ARTEFACTS_STORED, event.TaskProgressArtefactsStored},
		{"ACTION_FINALIZED", cascade.SupernodeEventType_ACTION_FINALIZED, event.TaskProgressActionFinalized},
		{"UNKNOWN_TYPE", cascade.SupernodeEventType(999), event.EventType("task.progress.unknown")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := toSdkEvent(tt.input)
			require.Equal(t, tt.expected, actual)
		})
	}
}
