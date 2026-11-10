package spawnapi

import (
	"context"
	"errors"
	"fmt"
	"slices"

	authenticationv1 "k8s.io/api/authentication/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Audience of the hub's projected token. The API server does not accept it, so
// the token is useless anywhere but here.
const Audience = "mip-notebook-operator"

// ErrUnauthenticated means the token is not valid for Audience.
var ErrUnauthenticated = errors.New("token is not valid for audience " + Audience)

// Authenticator maps a bearer token to a Kubernetes user name.
type Authenticator interface {
	// Authenticate returns ErrUnauthenticated for a token that is not valid,
	// another error when it cannot tell.
	Authenticate(ctx context.Context, token string) (string, error)
}

// TokenReviewAuthenticator asks the API server (tokenreviews create).
type TokenReviewAuthenticator struct {
	Client client.Client
}

func (a *TokenReviewAuthenticator) Authenticate(ctx context.Context, token string) (string, error) {
	tr := &authenticationv1.TokenReview{Spec: authenticationv1.TokenReviewSpec{
		Token:     token,
		Audiences: []string{Audience},
	}}
	if err := a.Client.Create(ctx, tr); err != nil {
		return "", fmt.Errorf("token review: %w", err)
	}
	// An authenticator without audience support may ignore spec.audiences;
	// status.audiences is what was actually checked.
	if !tr.Status.Authenticated || !slices.Contains(tr.Status.Audiences, Audience) {
		return "", ErrUnauthenticated
	}
	return tr.Status.User.Username, nil
}
