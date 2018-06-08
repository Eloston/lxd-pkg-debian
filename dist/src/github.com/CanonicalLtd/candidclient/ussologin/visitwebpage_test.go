// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package ussologin_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	"github.com/juju/usso"
	"golang.org/x/net/context"
	gc "gopkg.in/check.v1"
	errgo "gopkg.in/errgo.v1"
	"gopkg.in/httprequest.v1"
	"gopkg.in/macaroon-bakery.v2/httpbakery"

	"gopkg.in/CanonicalLtd/candidclient.v1/ussologin"
)

type interactorSuite struct {
	testing.CleanupSuite
}

var _ = gc.Suite(&interactorSuite{})

func (s *interactorSuite) TestKind(c *gc.C) {
	i := ussologin.NewInteractor(nil)
	c.Assert(i.Kind(), gc.Equals, "usso_oauth")
}

func (s *interactorSuite) TestInteractNotSupportedError(c *gc.C) {
	i := ussologin.NewInteractor(nil)
	req, err := http.NewRequest("GET", "", nil)
	c.Assert(err, gc.Equals, nil)
	ierr := httpbakery.NewInteractionRequiredError(nil, req)
	httpbakery.SetLegacyInteraction(ierr, "", "")
	_, err = i.Interact(context.Background(), nil, "", ierr)
	c.Assert(errgo.Cause(err), gc.Equals, httpbakery.ErrInteractionMethodNotFound)
}

func (s *interactorSuite) TestInteractGetTokenError(c *gc.C) {
	terr := errgo.New("test error")
	i := ussologin.NewInteractor(tokenGetterFunc(func(_ context.Context) (*usso.SSOData, error) {
		return nil, terr
	}))
	ierr := s.interactionRequiredError(c, "")
	_, err := i.Interact(context.Background(), nil, "", ierr)
	c.Assert(errgo.Cause(err), gc.Equals, terr)
}

func (s *interactorSuite) TestAuthenticatedRequest(c *gc.C) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Just check the request has a correct looking
		// Authorization header, we won't check the signature.
		c.Check(req.Header.Get("Authorization"), gc.Matches, "OAuth .*")
		httprequest.WriteJSON(w, http.StatusOK, ussologin.LoginResponse{
			DischargeToken: &httpbakery.DischargeToken{
				Kind:  "test",
				Value: []byte("test-token"),
			},
		})
	}))
	defer server.Close()

	i := ussologin.NewInteractor(tokenGetterFunc(func(_ context.Context) (*usso.SSOData, error) {
		return &usso.SSOData{
			ConsumerKey:    "test-user",
			ConsumerSecret: "test-user-secret",
			Realm:          "test",
			TokenKey:       "test-token",
			TokenName:      "test",
			TokenSecret:    "test-token-secret",
		}, nil
	}))
	ierr := s.interactionRequiredError(c, server.URL)
	dt, err := i.Interact(context.Background(), httpbakery.NewClient(), "", ierr)
	c.Assert(err, gc.Equals, nil)
	c.Assert(dt, jc.DeepEquals, &httpbakery.DischargeToken{
		Kind:  "test",
		Value: []byte("test-token"),
	})
}

func (s *interactorSuite) TestAuthenticatedRequestError(c *gc.C) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Just check the request has a correct looking
		// Authorization header, we won't check the signature.
		c.Check(req.Header.Get("Authorization"), gc.Matches, "OAuth .*")
		code, body := httpbakery.ErrorToResponse(context.Background(), errgo.New("test error"))
		httprequest.WriteJSON(w, code, body)
	}))
	defer server.Close()

	i := ussologin.NewInteractor(tokenGetterFunc(func(_ context.Context) (*usso.SSOData, error) {
		return &usso.SSOData{
			ConsumerKey:    "test-user",
			ConsumerSecret: "test-user-secret",
			Realm:          "test",
			TokenKey:       "test-token",
			TokenName:      "test",
			TokenSecret:    "test-token-secret",
		}, nil
	}))
	ierr := s.interactionRequiredError(c, server.URL)
	_, err := i.Interact(context.Background(), httpbakery.NewClient(), "", ierr)
	c.Assert(err, gc.ErrorMatches, `Get http.*: test error`)
}

func (s *interactorSuite) interactionRequiredError(c *gc.C, url string) *httpbakery.Error {
	req, err := http.NewRequest("GET", "", nil)
	c.Assert(err, gc.Equals, nil)
	ierr := httpbakery.NewInteractionRequiredError(nil, req)
	ussologin.SetInteraction(ierr, url)
	return ierr
}

type tokenGetterFunc func(ctx context.Context) (*usso.SSOData, error)

func (f tokenGetterFunc) GetToken(ctx context.Context) (*usso.SSOData, error) {
	return f(ctx)
}
