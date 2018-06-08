// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package ussodischarge_test

import (
	"net/http"
	"net/http/httptest"

	jc "github.com/juju/testing/checkers"
	"golang.org/x/net/context"
	gc "gopkg.in/check.v1"
	errgo "gopkg.in/errgo.v1"
	"gopkg.in/httprequest.v1"
	"gopkg.in/macaroon-bakery.v2/bakery"
	"gopkg.in/macaroon-bakery.v2/httpbakery"
	"gopkg.in/macaroon.v2"

	"gopkg.in/CanonicalLtd/candidclient.v1/params"
	"gopkg.in/CanonicalLtd/candidclient.v1/ussodischarge"
)

var _ httpbakery.Interactor = (*ussodischarge.Interactor)(nil)
var _ httpbakery.LegacyInteractor = (*ussodischarge.Interactor)(nil)

var testContext = context.Background()

type clientSuite struct {
	testMacaroon          *bakery.Macaroon
	testDischargeMacaroon *macaroon.Macaroon
	srv                   *httptest.Server

	// macaroon is returned from the /macaroon endpoint of the test server.
	// If this is nil, an error will be returned instead.
	macaroon *bakery.Macaroon
}

var _ = gc.Suite(&clientSuite{})

// ServeHTTP allows us to use the test suite as a handler to test the
// client methods against.
func (s *clientSuite) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/macaroon":
		s.serveMacaroon(w, r)
	case "/login":
		s.serveLogin(w, r)
	case "/api/v2/tokens/discharge":
		s.serveDischarge(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *clientSuite) SetUpTest(c *gc.C) {
	var err error
	s.testMacaroon, err = bakery.NewMacaroon([]byte("test rootkey"), []byte("test macaroon"), "test location", bakery.LatestVersion, nil)
	c.Assert(err, gc.Equals, nil)
	// Discharge macaroons from Ubuntu SSO will be binary encoded in the version 1 format.
	s.testDischargeMacaroon, err = macaroon.New([]byte("test discharge rootkey"), []byte("test discharge macaroon"), "test discharge location", macaroon.V1)
	c.Assert(err, gc.Equals, nil)

	s.srv = httptest.NewServer(s)
	s.macaroon = nil
}

func (s *clientSuite) TearDownTest(c *gc.C) {
	s.srv.Close()
}

func (s *clientSuite) TestMacaroon(c *gc.C) {
	s.macaroon = s.testMacaroon
	m, err := ussodischarge.Macaroon(testContext, nil, s.srv.URL+"/macaroon")
	c.Assert(err, gc.Equals, nil)
	c.Assert(m.M(), jc.DeepEquals, s.testMacaroon.M())
}

func (s *clientSuite) TestMacaroonError(c *gc.C) {
	m, err := ussodischarge.Macaroon(testContext, nil, s.srv.URL+"/macaroon")
	c.Assert(m, gc.IsNil)
	c.Assert(err, gc.ErrorMatches, `cannot get macaroon: Get http.*: test error`)
}

func (s *clientSuite) TestVisitor(c *gc.C) {
	v := ussodischarge.NewInteractor(func(_ *httpbakery.Client, url string) (macaroon.Slice, error) {
		c.Assert(url, gc.Equals, s.srv.URL+"/login")
		return macaroon.Slice{s.testMacaroon.M()}, nil
	})

	client := httpbakery.NewClient()
	req, err := http.NewRequest("GET", "", nil)
	c.Assert(err, gc.Equals, nil)
	ierr := httpbakery.NewInteractionRequiredError(nil, req)
	ussodischarge.SetInteraction(ierr, s.srv.URL+"/login")
	dt, err := v.Interact(testContext, client, "", ierr)
	c.Assert(err, gc.Equals, nil)
	c.Assert(dt, jc.DeepEquals, &httpbakery.DischargeToken{
		Kind:  "test-kind",
		Value: []byte("test-value"),
	})
}

func (s *clientSuite) TestVisitorMethodNotSupported(c *gc.C) {
	v := ussodischarge.NewInteractor(func(_ *httpbakery.Client, url string) (macaroon.Slice, error) {
		return nil, errgo.New("function called unexpectedly")
	})
	client := httpbakery.NewClient()
	req, err := http.NewRequest("GET", "", nil)
	c.Assert(err, gc.Equals, nil)
	ierr := httpbakery.NewInteractionRequiredError(nil, req)
	ierr.SetInteraction("other", nil)
	dt, err := v.Interact(testContext, client, "", ierr)
	c.Assert(errgo.Cause(err), gc.Equals, httpbakery.ErrInteractionMethodNotFound)
	c.Assert(dt, gc.IsNil)
}

func (s *clientSuite) TestVisitorFunctionError(c *gc.C) {
	v := ussodischarge.NewInteractor(func(_ *httpbakery.Client, url string) (macaroon.Slice, error) {
		return nil, errgo.WithCausef(nil, testCause, "test error")
	})
	client := httpbakery.NewClient()
	req, err := http.NewRequest("GET", "", nil)
	c.Assert(err, gc.Equals, nil)
	ierr := httpbakery.NewInteractionRequiredError(nil, req)
	ussodischarge.SetInteraction(ierr, s.srv.URL+"/login")
	dt, err := v.Interact(testContext, client, "", ierr)
	c.Assert(errgo.Cause(err), gc.Equals, testCause)
	c.Assert(err, gc.ErrorMatches, "test error")
	c.Assert(dt, gc.IsNil)
}

func (s *clientSuite) TestAcquireDischarge(c *gc.C) {
	d := &ussodischarge.Discharger{
		Email:    "user@example.com",
		Password: "secret",
		OTP:      "123456",
	}
	m, err := d.AcquireDischarge(testContext, macaroon.Caveat{
		Location: s.srv.URL,
		Id:       []byte("test caveat id"),
	}, nil)
	c.Assert(err, gc.Equals, nil)
	c.Assert(m.M(), jc.DeepEquals, s.testDischargeMacaroon)
}

func (s *clientSuite) TestAcquireDischargeError(c *gc.C) {
	d := &ussodischarge.Discharger{
		Email:    "user@example.com",
		Password: "bad-secret",
		OTP:      "123456",
	}
	m, err := d.AcquireDischarge(testContext, macaroon.Caveat{
		Location: s.srv.URL,
		Id:       []byte("test caveat id"),
	}, nil)
	c.Assert(err, gc.ErrorMatches, `Post http.*: Provided email/password is not correct.`)
	c.Assert(m, gc.IsNil)
}

func (s *clientSuite) TestDischargeAll(c *gc.C) {
	m := s.testMacaroon.Clone()
	err := m.M().AddThirdPartyCaveat([]byte("third party root key"), []byte("third party caveat id"), s.srv.URL)
	c.Assert(err, gc.Equals, nil)
	d := &ussodischarge.Discharger{
		Email:    "user@example.com",
		Password: "secret",
		OTP:      "123456",
	}
	ms, err := d.DischargeAll(testContext, m)
	c.Assert(err, gc.Equals, nil)
	md := s.testDischargeMacaroon.Clone()
	md.Bind(m.M().Signature())
	c.Assert(ms, jc.DeepEquals, macaroon.Slice{m.M(), md})
}

func (s *clientSuite) TestDischargeAllError(c *gc.C) {
	m := s.testMacaroon.Clone()
	err := m.M().AddThirdPartyCaveat([]byte("third party root key"), []byte("third party caveat id"), s.srv.URL)
	c.Assert(err, gc.Equals, nil)
	d := &ussodischarge.Discharger{
		Email:    "user@example.com",
		Password: "bad-secret",
		OTP:      "123456",
	}
	ms, err := d.DischargeAll(testContext, m)
	c.Assert(err, gc.ErrorMatches, `cannot get discharge from ".*": Post http.*: Provided email/password is not correct.`)
	c.Assert(ms, gc.IsNil)
}

func (s *clientSuite) serveMacaroon(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		fail(w, r, errgo.Newf("bad method: %s", r.Method))
	}
	if s.macaroon != nil {
		httprequest.WriteJSON(w, http.StatusOK, ussodischarge.MacaroonResponse{
			Macaroon: s.macaroon,
		})
	} else {
		httprequest.WriteJSON(w, http.StatusInternalServerError, params.Error{
			Message: "test error",
		})
	}
}

func (s *clientSuite) serveLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		fail(w, r, errgo.Newf("bad method: %s", r.Method))
	}
	var lr ussodischarge.LoginRequest
	if err := httprequest.Unmarshal(httprequest.Params{Request: r, Response: w}, &lr); err != nil {
		fail(w, r, err)
	}
	if n := len(lr.Login.Macaroons); n != 1 {
		fail(w, r, errgo.Newf("macaroon slice has unexpected length %d", n))
	}
	if id := lr.Login.Macaroons[0].Id(); string(id) != "test macaroon" {
		fail(w, r, errgo.Newf("unexpected macaroon sent %q", string(id)))
	}
	httprequest.WriteJSON(w, http.StatusOK, ussodischarge.LoginResponse{
		DischargeToken: &httpbakery.DischargeToken{
			Kind:  "test-kind",
			Value: []byte("test-value"),
		},
	})
}

func (s *clientSuite) serveDischarge(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		fail(w, r, errgo.Newf("bad method: %s", r.Method))
	}
	var dr ussodischarge.USSODischargeRequest
	if err := httprequest.Unmarshal(httprequest.Params{Request: r, Response: w}, &dr); err != nil {
		fail(w, r, err)
	}
	if dr.Discharge.Email == "" {
		fail(w, r, errgo.New("email not specified"))
	}
	if dr.Discharge.Password == "" {
		fail(w, r, errgo.New("password not specified"))
	}
	if dr.Discharge.OTP == "" {
		fail(w, r, errgo.New("otp not specified"))
	}
	if dr.Discharge.CaveatID == "" {
		fail(w, r, errgo.New("caveat_id not specified"))
	}
	if dr.Discharge.Email != "user@example.com" || dr.Discharge.Password != "secret" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error_list": [{"message": "Provided email/password is not correct.", "code": "invalid-credentials"}], "message": "Provided email/password is not correct.", "code": "INVALID_CREDENTIALS", "extra": {}}`))
		return
	}
	var m ussodischarge.USSOMacaroon
	m.Macaroon = *s.testDischargeMacaroon
	httprequest.WriteJSON(w, http.StatusOK, map[string]interface{}{"discharge_macaroon": &m})
}

func fail(w http.ResponseWriter, r *http.Request, err error) {
	httprequest.WriteJSON(w, http.StatusBadRequest, params.Error{
		Message: err.Error(),
	})
}

var testCause = errgo.New("test cause")
