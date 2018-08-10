package resourceadapter

import (
	"github.com/cphrn/go/services/horizon/internal/db2/core"
	. "github.com/cphrn/go/protocols/horizon"
)

func PopulateAccountThresholds(dest *AccountThresholds, row core.Account) {
	dest.LowThreshold = row.Thresholds[1]
	dest.MedThreshold = row.Thresholds[2]
	dest.HighThreshold = row.Thresholds[3]
}
