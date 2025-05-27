package api

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/sigstore/sigstore-go/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature/payload"
)

const (
	GetAttestationByRepoAndSubjectDigestPath  = "repos/%s/attestations/%s"
	GetAttestationByOwnerAndSubjectDigestPath = "orgs/%s/attestations/%s"
)

var ErrNoAttestationsFound = errors.New("no attestations found")

type Attestation struct {
	Bundle    *signature.VerificationBundle `json:"bundle"`
	BundleURL string                          `json:"bundle_url"`
}

type AttestationsResponse struct {
	Attestations []*Attestation `json:"attestations"`
}

type IntotoStatement struct {
	PredicateType string `json:"predicateType"`
}

func FilterAttestations(predicateType string, attestations []*Attestation) ([]*Attestation, error) {
	filteredAttestations := []*Attestation{}

	for _, each := range attestations {
		env, err := each.Bundle.GetEnvelope()
		if err != nil {
			continue // or handle the error appropriately
		}
		if env != nil {
			payloadType, err := env.PayloadType()
			if err != nil {
				continue
			}
			if payloadType != payload.MimeTypeIntotoPayload {
				// Don't fail just because an entry isn't intoto
				continue
			}
			payloadBytes, err := env.Payload()
			if err != nil {
				continue
			}
			var intotoStatement IntotoStatement
			if err := json.Unmarshal(payloadBytes, &intotoStatement); err != nil {
				// Don't fail just because a single entry can't be unmarshalled
				continue
			}
			if intotoStatement.PredicateType == predicateType {
				filteredAttestations = append(filteredAttestations, each)
			}
		}
	}

	if len(filteredAttestations) == 0 {
		return nil, fmt.Errorf("no attestations found with predicate type: %s", predicateType)
	}

	return filteredAttestations, nil
}