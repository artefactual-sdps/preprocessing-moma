package activities

import (
	"context"
)

const SipCreationName = "sip-creation"

type SipCreationActivity struct{}

func NewSipCreationActivity() *SipCreationActivity {
	return &SipCreationActivity{}
}

type SipCreationParams struct {
	SipPath string
}

type SipCreationResult struct {
	Out        string
	NewSipPath string
}

func (sc *SipCreationActivity) Execute(ctx context.Context, params *SipCreationParams) (*SipCreationResult, error) {
	// TODO change this activity into a bag activity
	res := &SipCreationResult{}

	res.NewSipPath = params.SipPath + "_bag"
	return res, nil
}
