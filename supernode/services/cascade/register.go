package cascade

import (
	"context"
	"os"

	"github.com/LumeraProtocol/supernode/v2/pkg/logtrace"
	"github.com/LumeraProtocol/supernode/v2/supernode/services/common"
)

// RegisterRequest contains parameters for upload request
type RegisterRequest struct {
	TaskID   string
	ActionID string
	DataHash []byte
	DataSize int
	FilePath string
}

// RegisterResponse contains the result of upload
type RegisterResponse struct {
	EventType SupernodeEventType
	Message   string
	TxHash    string
}

// Register processes the upload request for cascade input data.
// 1- Fetch & validate action (it should be a cascade action registered on the chain)
// 2- Ensure this super-node is eligible to process the action (should be in the top supernodes list for the action block height)
// 3- Get the cascade metadata from the action: it contains the data hash and the signatures
//
//	Assuming data hash is a base64 encoded string of blake3 hash of the data
//	The signatures field is: b64(JSON(Layout)).Signature where Layout is codec.Layout
//	The layout is a JSON object that contains the metadata of the data
//
// 4- Verify the data hash (the data hash should match the one in the action ticket) - again, hash function should be blake3
// 5- Generate Symbols with codec (RQ-Go Library) (the data should be encoded using the codec)
// 6- Extract the layout and the signature from Step 3. Verify the signature using the creator's public key (creator address is in the action)
// 7- Generate RQ-ID files from the layout that we generated locally and then match those with the ones in the action
// 8- Verify the IDs in the layout and the metadata (the IDs should match the ones in the action)
// 9- Store the artefacts in P2P Storage (the redundant metadata files and the symbols from the symbols dir)
func (task *CascadeRegistrationTask) Register(
	ctx context.Context,
	req *RegisterRequest,
	send func(resp *RegisterResponse) error,
) (err error) {

	// Use a server-owned context for long-running operations (storage/finalization)
	// to avoid being bound to the gRPC stream deadline/cancellation.
	// taskCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	// defer cancel()

	fields := logtrace.Fields{logtrace.FieldMethod: "Register", logtrace.FieldRequest: req}
	logtrace.Info(ctx, "cascade-action-registration request received", fields)

	// Ensure task status and resources are finalized regardless of outcome
	defer func() {
		if err != nil {
			task.UpdateStatus(common.StatusTaskCanceled)
		} else {
			task.UpdateStatus(common.StatusTaskCompleted)
		}
		task.Cancel()
	}()

	// Always attempt to remove the uploaded file path
	defer func() {
		if req != nil && req.FilePath != "" {
			if remErr := os.RemoveAll(req.FilePath); remErr != nil {
				logtrace.Warn(ctx, "error removing file", fields)
			} else {
				logtrace.Info(ctx, "input file has been cleaned up", fields)
			}
		}
	}()

	/* 1. Fetch & validate action -------------------------------------------------- */
	action, err := task.fetchAction(ctx, req.ActionID, fields)
	if err != nil {
		return err
	}
	fields[logtrace.FieldBlockHeight] = action.BlockHeight
	fields[logtrace.FieldCreator] = action.Creator
	fields[logtrace.FieldStatus] = action.State
	fields[logtrace.FieldPrice] = action.Price
	logtrace.Info(ctx, "action has been retrieved", fields)
	task.streamEvent(SupernodeEventTypeActionRetrieved, "action has been retrieved", "", send)

	/* 2. Verify action fee -------------------------------------------------------- */
	if err := task.verifyActionFee(ctx, action, req.DataSize, fields); err != nil {
		return err
	}
	logtrace.Info(ctx, "action fee has been validated", fields)
	task.streamEvent(SupernodeEventTypeActionFeeVerified, "action-fee has been validated", "", send)

	/* 3. Ensure this super-node is eligible -------------------------------------- */
	fields[logtrace.FieldSupernodeState] = task.config.SupernodeAccountAddress
	if err := task.ensureIsTopSupernode(ctx, uint64(action.BlockHeight), fields); err != nil {
		return err
	}
	logtrace.Info(ctx, "current-supernode exists in the top-sn list", fields)
	task.streamEvent(SupernodeEventTypeTopSupernodeCheckPassed, "current supernode exists in the top-sn list", "", send)

	/* 4. Decode cascade metadata -------------------------------------------------- */
	cascadeMeta, err := task.decodeCascadeMetadata(ctx, action.Metadata, fields)
	if err != nil {
		return err
	}
	logtrace.Info(ctx, "cascade metadata decoded", fields)
	task.streamEvent(SupernodeEventTypeMetadataDecoded, "cascade metadata has been decoded", "", send)

	/* 5. Verify data hash --------------------------------------------------------- */
	if err := task.verifyDataHash(ctx, req.DataHash, cascadeMeta.DataHash, fields); err != nil {
		return err
	}
	logtrace.Info(ctx, "data-hash has been verified", fields)
	task.streamEvent(SupernodeEventTypeDataHashVerified, "data-hash has been verified", "", send)

	/* 6. Encode the raw data ------------------------------------------------------ */
	encResp, err := task.encodeInput(ctx, req.ActionID, req.FilePath, req.DataSize, fields)
	if err != nil {
		return err
	}
	logtrace.Info(ctx, "input-data has been encoded", fields)
	task.streamEvent(SupernodeEventTypeInputEncoded, "input data has been encoded", "", send)

	/* 7. Signature verification + layout decode ---------------------------------- */
	layout, signature, err := task.verifySignatureAndDecodeLayout(
		ctx, cascadeMeta.Signatures, action.Creator, encResp.Metadata, fields,
	)
	if err != nil {
		return err
	}
	logtrace.Info(ctx, "signature has been verified", fields)
	task.streamEvent(SupernodeEventTypeSignatureVerified, "signature has been verified", "", send)

	/* 8. Generate RQ-ID files ----------------------------------------------------- */
	rqidResp, err := task.generateRQIDFiles(ctx, cascadeMeta, signature, action.Creator, encResp.Metadata, fields)
	if err != nil {
		return err
	}
	logtrace.Info(ctx, "rq-id files have been generated", fields)
	task.streamEvent(SupernodeEventTypeRQIDsGenerated, "rq-id files have been generated", "", send)

	/* 9. Consistency checks ------------------------------------------------------- */
	if err := verifyIDs(layout, encResp.Metadata); err != nil {
		return task.wrapErr(ctx, "failed to verify IDs", err, fields)
	}
	logtrace.Info(ctx, "rq-ids have been verified", fields)
	task.streamEvent(SupernodeEventTypeRqIDsVerified, "rq-ids have been verified", "", send)

	/* 10. Persist artefacts -------------------------------------------------------- */
	// Store artefacts (ID files + symbols) using task-scoped context so it can
	// proceed even if the client disconnects/cancels the stream.
	if err := task.storeArtefacts(ctx, action.ActionID, rqidResp.RedundantMetadataFiles, encResp.SymbolsDir, fields); err != nil {
		return err
	}
	logtrace.Info(ctx, "artefacts have been stored", fields)
	task.streamEvent(SupernodeEventTypeArtefactsStored, "artefacts have been stored", "", send)

	// Finalize on-chain using the task context as well, to avoid premature
	// cancellation from the client stream context.
	resp, err := task.LumeraClient.FinalizeAction(ctx, action.ActionID, rqidResp.RQIDs)
	if err != nil {
		fields[logtrace.FieldError] = err.Error()
		logtrace.Info(ctx, "Finalize Action Error", fields)
		return task.wrapErr(ctx, "failed to finalize action", err, fields)
	}
	txHash := resp.TxResponse.TxHash
	fields[logtrace.FieldTxHash] = txHash
	logtrace.Info(ctx, "action has been finalized", fields)
	task.streamEvent(SupernodeEventTypeActionFinalized, "action has been finalized", txHash, send)

	return nil
}
