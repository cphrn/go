package horizon

import (
	"github.com/cphrn/go/services/horizon/internal/ledger"
	"github.com/cphrn/go/services/horizon/internal/resourceadapter"
	"github.com/cphrn/go/protocols/horizon"
	"github.com/cphrn/go/support/render/hal"
)

// RootAction provides a summary of the horizon instance and links to various
// useful endpoints
type RootAction struct {
	Action
}

// JSON renders the json response for RootAction
func (action *RootAction) JSON() {
	var res horizon.Root
	resourceadapter.PopulateRoot(
		action.R.Context(),
		&res,
		ledger.CurrentState(),
		action.App.horizonVersion,
		action.App.coreVersion,
		action.App.networkPassphrase,
		action.App.protocolVersion,
	)

	hal.Render(action.W, res)
}
