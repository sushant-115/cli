package verification

import (
	"fmt"
	"testing"

	"github.com/cli/cli/v2/pkg/cmd/attestation/api"
	"github.com/cli/cli/v2/pkg/cmd/attestation/test/data"

	in_toto "github.com/in-toto/attestation/go/v1"
	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/verify"
	"github.com/sigstore/sigstore-go/pkg/verify/policy"
)

type MockSigstoreVerifier struct {
	t           *testing.T
	mockResults []*AttestationProcessingResult
}

func (v *MockSigstoreVerifier) Verify([]*api.Attestation, policy.Options) ([]*AttestationProcessingResult, error) {
	if v.mockResults != nil {
		return v.mockResults, nil
	}

	statement := &in_toto.Statement{}
	statement.PredicateType = SLSAPredicateV1

	result := AttestationProcessingResult{
		Attestation: &api.Attestation{
			Bundle: data.SigstoreBundle(v.t),
		},
		VerificationResult: &verify.VerificationResult{
			Statement: statement,
			// Signature: &verify.SignatureVerificationResult{ // Removed struct and inlined fields
			CertInfo: &verify.CertInfo{
				BuildSignerURI:           "https://github.com/github/example/.github/workflows/release.yml@refs/heads/main",
				SourceRepositoryOwnerURI: "https://github.com/sigstore",
				SourceRepositoryURI:      "https://github.com/sigstore/sigstore-js",
				Issuer:                   "https://token.actions.githubusercontent.com",
			},
			// },
		},
	}

	results := []*AttestationProcessingResult{&result}

	return results, nil
}

func NewMockSigstoreVerifier(t *testing.T) *MockSigstoreVerifier {
	result := BuildSigstoreJsMockResult(t)
	results := []*AttestationProcessingResult{&result}

	return &MockSigstoreVerifier{t, results}
}

func NewMockSigstoreVerifierWithMockResults(t *testing.T, mockResults []*AttestationProcessingResult) *MockSigstoreVerifier {
	return &MockSigstoreVerifier{t, mockResults}
}

type FailSigstoreVerifier struct{}

func (v *FailSigstoreVerifier) Verify([]*api.Attestation, policy.Options) ([]*AttestationProcessingResult, error) {
	return nil, fmt.Errorf("failed to verify attestations")
}

func BuildMockResult(b *bundle.Bundle, buildConfigURI, buildSignerURI, sourceRepoOwnerURI, sourceRepoURI, issuer string) AttestationProcessingResult {
	statement := &in_toto.Statement{}
	statement.PredicateType = SLSAPredicateV1

	return AttestationProcessingResult{
		Attestation: &api.Attestation{
			Bundle: b,
		},
		VerificationResult: &verify.VerificationResult{
			Statement: statement,
			// Signature: &verify.SignatureVerificationResult{ // Removed struct and inlined fields
			CertInfo: &verify.CertInfo{
				BuildConfigURI:           buildConfigURI,
				BuildSignerURI:           buildSignerURI,
				Issuer:                   issuer,
				SourceRepositoryOwnerURI: sourceRepoOwnerURI,
				SourceRepositoryURI:      sourceRepoURI,
			},
			// },
		},
	}
}

func BuildSigstoreJsMockResult(t *testing.T) AttestationProcessingResult {
	bundle := data.SigstoreBundle(t)
	buildConfigURI := "https://github.com/sigstore/sigstore-js/.github/workflows/build.yml@refs/heads/main"
	buildSignerURI := "https://github.com/github/example/.github/workflows/release.yml@refs/heads/main"
	sourceRepoOwnerURI := "https://github.com/sigstore"
	sourceRepoURI := "https://github.com/sigstore/sigstore-js"
	issuer := "https://token.actions.githubusercontent.com"
	return BuildMockResult(bundle, buildConfigURI, buildSignerURI, sourceRepoOwnerURI, sourceRepoURI, issuer)
}