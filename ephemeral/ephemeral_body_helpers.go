package ephemeral

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/magodo/terraform-plugin-framework-helper/dynamic"
	"github.com/magodo/terraform-plugin-framework-helper/jsonset"
)

// ValidateEphemeralBody validates a known, non-null ephemeral_body doesn't joint with the body.
// It returns the json representation of the ephemeral body as well (if known, non-null).
func ValidateEphemeralBody(body []byte, ephemeralBody types.Dynamic) ([]byte, diag.Diagnostics) {
	if ephemeralBody.IsUnknown() || ephemeralBody.IsNull() {
		return nil, nil
	}

	var diags diag.Diagnostics

	eb, err := dynamic.ToJSON(ephemeralBody)
	if err != nil {
		diags.AddError(
			"Invalid configuration",
			fmt.Sprintf(`marshal "ephemeral_body": %v`, err),
		)
		return nil, diags
	}
	disjointed, err := jsonset.Disjointed(body, eb)
	if err != nil {
		diags.AddError(
			"Invalid configuration",
			fmt.Sprintf(`checking disjoint of "body" and "ephemeral_body": %v`, err),
		)
		return nil, diags
	}
	if !disjointed {
		diags.AddError(
			"Invalid configuration",
			`"body" and "ephemeral_body" are not disjointed`,
		)
		return nil, diags
	}
	return eb, nil
}
