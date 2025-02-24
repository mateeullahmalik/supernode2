package lumera

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/types"
)

func (c *Client) Sign(ctx context.Context, senderName string, msgs []types.Msg) ([]byte, error) {
	txf := tx.Factory{}.
		WithKeybase(c.cosmosSdk.Keyring).
		WithChainID(c.cosmosSdk.ChainID)

	txBuilder, err := txf.BuildUnsignedTx(msgs...)
	if err != nil {
		return nil, fmt.Errorf("build tx error: %w", err)
	}

	if err := tx.Sign(ctx, txf, senderName, txBuilder, true); err != nil {
		return nil, fmt.Errorf("sign tx error: %w", err)
	}

	txBytes, err := c.cosmosSdk.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return nil, fmt.Errorf("encode tx error: %w", err)
	}

	return txBytes, nil
}
