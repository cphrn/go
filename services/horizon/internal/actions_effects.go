package horizon

import (
	"fmt"
	"regexp"

	"github.com/cphrn/go/services/horizon/internal/db2"
	"github.com/cphrn/go/services/horizon/internal/db2/history"
	"github.com/cphrn/go/support/errors"
	"github.com/cphrn/go/support/render/hal"
	"github.com/cphrn/go/services/horizon/internal/render/sse"
	"github.com/cphrn/go/services/horizon/internal/resourceadapter"
)

// This file contains the actions:
//
// EffectIndexAction: pages of effects

// EffectIndexAction renders a page of effect resources, identified by
// a normal page query and optionally filtered by an account, ledger,
// transaction, or operation.
type EffectIndexAction struct {
	Action
	AccountFilter     string
	LedgerFilter      int32
	TransactionFilter string
	OperationFilter   int64

	PagingParams db2.PageQuery
	Records      []history.Effect
	Page         hal.Page
	Ledgers      history.LedgerCache
}

// JSON is a method for actions.JSON
func (action *EffectIndexAction) JSON() {
	action.Do(
		action.EnsureHistoryFreshness,
		action.loadParams,
		action.ValidateCursorWithinHistory,
		action.loadRecords,
		action.loadLedgers,
		action.loadPage,
	)

	action.Do(func() {
		hal.Render(action.W, action.Page)
	})
}

// SSE is a method for actions.SSE
func (action *EffectIndexAction) SSE(stream sse.Stream) {
	action.Setup(
		action.EnsureHistoryFreshness,
		action.loadParams,
		action.ValidateCursorWithinHistory,
	)

	action.Do(
		action.loadRecords,
		action.loadLedgers,
		func() {
			stream.SetLimit(int(action.PagingParams.Limit))
			records := action.Records[stream.SentCount():]

			for _, record := range records {
				ledger, found := action.Ledgers.Records[record.LedgerSequence()]
				if !found {
					msg := fmt.Sprintf("could not find ledger data for sequence %d", record.LedgerSequence())
					stream.Err(errors.New(msg))
					return
				}

				res, err := resourceadapter.NewEffect(action.R.Context(), record, ledger)

				if err != nil {
					stream.Err(action.Err)
					return
				}

				stream.Send(sse.Event{
					ID:   res.PagingToken(),
					Data: res,
				})
			}
		},
	)
}

// loadLedgers populates the ledger cache for this action
func (action *EffectIndexAction) loadLedgers() {
	for _, eff := range action.Records {
		action.Ledgers.Queue(eff.LedgerSequence())
	}

	action.Err = action.Ledgers.Load(action.HistoryQ())
}

func (action *EffectIndexAction) loadParams() {
	action.ValidateCursor()
	action.PagingParams = action.GetPageQuery()
	action.AccountFilter = action.GetString("account_id")
	action.LedgerFilter = action.GetInt32("ledger_id")
	action.TransactionFilter = action.GetString("tx_id")
	action.OperationFilter = action.GetInt64("op_id")
}

// loadRecords populates action.Records
func (action *EffectIndexAction) loadRecords() {
	effects := action.HistoryQ().Effects()

	switch {
	case action.AccountFilter != "":
		effects.ForAccount(action.AccountFilter)
	case action.LedgerFilter > 0:
		effects.ForLedger(action.LedgerFilter)
	case action.OperationFilter > 0:
		effects.ForOperation(action.OperationFilter)
	case action.TransactionFilter != "":
		effects.ForTransaction(action.TransactionFilter)
	}

	action.Err = effects.Page(action.PagingParams).Select(&action.Records)
}

// loadPage populates action.Page
func (action *EffectIndexAction) loadPage() {
	for _, record := range action.Records {

		ledger, found := action.Ledgers.Records[record.LedgerSequence()]
		if !found {
			msg := fmt.Sprintf("could not find ledger data for sequence %d", record.LedgerSequence())
			action.Err = errors.New(msg)
			return
		}

		var res hal.Pageable
		res, action.Err = resourceadapter.NewEffect(action.R.Context(), record, ledger)
		if action.Err != nil {
			return
		}
		action.Page.Add(res)
	}

	action.Page.FullURL = action.FullURL()
	action.Page.Limit = action.PagingParams.Limit
	action.Page.Cursor = action.PagingParams.Cursor
	action.Page.Order = action.PagingParams.Order
	action.Page.PopulateLinks()
}

// ValidateCursor ensures that the provided cursor parameter is of the form
// OPERATIONID-INDEX (such as 1234-56) or is the special value "now" that
// represents the the cursor directly after the last closed ledger
func (action *EffectIndexAction) ValidateCursor() {
	c := action.GetString("cursor")

	if c == "" {
		return
	}

	ok, err := regexp.MatchString("now|\\d+(-\\d+)?", c)
	if err != nil {
		action.Err = err
		return
	}

	if !ok {
		action.SetInvalidField("cursor", errors.New("invalid format"))
		return
	}

	return
}
