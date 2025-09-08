package ephemeral

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/magodo/terraform-plugin-framework-helper/dynamic"
	"github.com/magodo/terraform-plugin-framework-helper/jsonset"
)

const (
	pkEphemeralBody = "ephemeral_body"
)

type PrivateData interface {
	GetKey(ctx context.Context, key string) ([]byte, diag.Diagnostics)
	SetKey(ctx context.Context, key string, value []byte) diag.Diagnostics
}

func Exists(ctx context.Context, d PrivateData) (bool, diag.Diagnostics) {
	b, diags := d.GetKey(ctx, pkEphemeralBody)
	if diags.HasError() {
		return false, diags
	}
	return b != nil, diags
}

// Set sets the hash of the ephemeral body to the private state.
// If `ebody` is nil, it removes the hash from the private state.
func Set(ctx context.Context, d PrivateData, ebody []byte) (diags diag.Diagnostics) {
	if ebody == nil {
		d.SetKey(ctx, pkEphemeralBody, nil)
		return
	}

	// Calculate the hash of the ephemeral body
	h := sha256.New()
	if _, err := h.Write(ebody); err != nil {
		diags.AddError(
			`Error to hash the ephemeral body`,
			err.Error(),
		)
		return
	}
	hash := h.Sum(nil)

	// Nullify ephemeral body
	nb, err := jsonset.NullifyObject(ebody)
	if err != nil {
		diags.AddError(
			`Error to nullify the ephemeral body`,
			err.Error(),
		)
		return
	}

	b, err := json.Marshal(map[string]interface{}{
		// []byte will be marshaled to base64 encoded string
		"hash": hash,
		"null": nb,
	})
	if err != nil {
		diags.AddError(
			`Error to marshal the ephemeral body private data`,
			err.Error(),
		)
		return
	}

	return d.SetKey(ctx, pkEphemeralBody, b)
}

// Diff tells whether the ephemeral body is different than the hash stored in the private state.
// In case private state doesn't have the record, regard the record as "nil" (i.e. will return true if ebody is non-nil).
// In case private state has the record (guaranteed to be non-nil), while ebody is nil, it also returns true.
func Diff(ctx context.Context, d PrivateData, ephemeralBody types.Dynamic) (bool, diag.Diagnostics) {
	if ephemeralBody.IsUnknown() {
		return true, nil
	}

	b, diags := d.GetKey(ctx, pkEphemeralBody)
	if diags.HasError() {
		return false, diags
	}
	if b == nil {
		// In case private state doesn't store the key yet, it only diffs when the ebody is not nil.
		return !ephemeralBody.IsNull(), diags
	}

	// Calc the hash in the private data
	var mm map[string]interface{}
	if err := json.Unmarshal(b, &mm); err != nil {
		diags.AddError(
			`Error to unmarshal the ephemeral body private data`,
			err.Error(),
		)
		return false, diags
	}
	privateHashEnc, ok := mm["hash"]
	if !ok {
		diags.AddError(
			`Invalid ephemeral body private data`,
			`Key "hash" not found`,
		)
		return false, diags
	}

	// Calc the hash of the ebody
	ebody, err := dynamic.ToJSON(ephemeralBody)
	if err != nil {
		diags.AddError(
			`Error to marshal the ephemeral body`,
			err.Error(),
		)
		return false, diags
	}
	h := sha256.New()
	if _, err := h.Write(ebody); err != nil {
		diags.AddError(
			`Error to hash ephemeral body`,
			err.Error(),
		)
		return false, diags
	}
	hash := h.Sum(nil)
	hashEnc := base64.StdEncoding.EncodeToString(hash)

	return hashEnc != privateHashEnc.(string), diags
}

// GetNullBody gets the nullified ephemeral body from the private data.
// If it doesn't exist, nil is returned.
func GetNullBody(ctx context.Context, d PrivateData) ([]byte, diag.Diagnostics) {
	b, diags := d.GetKey(ctx, pkEphemeralBody)
	if diags.HasError() {
		return nil, diags
	}
	if b == nil {
		return nil, nil
	}

	var mm map[string]interface{}
	if err := json.Unmarshal(b, &mm); err != nil {
		diags.AddError(
			`Error to unmarshal the ephemeral body private data`,
			err.Error(),
		)
		return nil, diags
	}
	bEnc, ok := mm["null"]
	if !ok {
		return nil, nil
	}
	b, err := base64.StdEncoding.DecodeString(bEnc.(string))
	if err != nil {
		diags.AddError(
			`Error base64 decoding the nullified the ephemeral body in the private data`,
			err.Error(),
		)
		return nil, diags
	}
	return b, nil
}

// ValidateEphemeralBody validates a known, non-null ephemeral body doesn't joint with the body.
// It returns the json representation of the ephemeral body as well (if known, non-null).
func ValidateEphemeralBody(body []byte, ephemeralBody types.Dynamic) ([]byte, diag.Diagnostics) {
	if ephemeralBody.IsUnknown() || ephemeralBody.IsNull() {
		return nil, nil
	}

	var diags diag.Diagnostics

	eb, err := dynamic.ToJSON(ephemeralBody)
	if err != nil {
		diags.AddError(
			"failed to marshal ephemeral body",
			err.Error(),
		)
		return nil, diags
	}
	disjointed, err := jsonset.Disjointed(body, eb)
	if err != nil {
		diags.AddError(
			"failed to check disjoint of the body and the ephemeral body",
			err.Error(),
		)
		return nil, diags
	}
	if !disjointed {
		diags.AddError(
			"the body and the ephemeral body are not disjointed",
			"",
		)
		return nil, diags
	}
	return eb, nil
}
