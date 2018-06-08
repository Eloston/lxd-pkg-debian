// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package ussologin_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	"github.com/juju/usso"
	"golang.org/x/net/context"
	gc "gopkg.in/check.v1"
	errgo "gopkg.in/errgo.v1"
	"gopkg.in/juju/environschema.v1/form"

	"gopkg.in/CanonicalLtd/candidclient.v1/ussologin"
)

type storeSuite struct {
	testing.CleanupSuite
}

var _ = gc.Suite(&storeSuite{})

func (s *storeSuite) TestPutGetToken(c *gc.C) {
	token := &usso.SSOData{
		ConsumerKey:    "consumerkey",
		ConsumerSecret: "consumersecret",
		Realm:          "realm",
		TokenKey:       "tokenkey",
		TokenName:      "tokenname",
		TokenSecret:    "tokensecret",
	}
	path := filepath.Join(c.MkDir(), "subdir", "tokenFile")
	store := ussologin.NewFileTokenStore(path)
	err := store.Put(token)
	c.Assert(err, jc.ErrorIsNil)

	tok, err := store.Get()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(tok, gc.DeepEquals, token)
	data, err := ioutil.ReadFile(path)
	c.Assert(err, jc.ErrorIsNil)
	var storedToken *usso.SSOData
	err = json.Unmarshal(data, &storedToken)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(token, gc.DeepEquals, storedToken)
}

func (s *storeSuite) TestReadInvalidToken(c *gc.C) {
	path := fmt.Sprintf("%s/tokenFile", c.MkDir())
	err := ioutil.WriteFile(path, []byte("foobar"), 0700)
	c.Assert(err, jc.ErrorIsNil)
	store := ussologin.NewFileTokenStore(path)

	_, err = store.Get()
	c.Assert(err, gc.ErrorMatches, `cannot unmarshal token: invalid character 'o' in literal false \(expecting 'a'\)`)
}

type storeTokenGetterSuite struct {
	testing.CleanupSuite
}

var _ = gc.Suite(&storeTokenGetterSuite{})

func (s *storeTokenGetterSuite) TestTokenInStore(c *gc.C) {
	testToken := &usso.SSOData{
		ConsumerKey:    "consumerkey",
		ConsumerSecret: "consumersecret",
		Realm:          "realm",
		TokenKey:       "tokenkey",
		TokenName:      "tokenname",
		TokenSecret:    "tokensecret",
	}
	st := &testTokenStore{
		tok: testToken,
	}
	g := &ussologin.StoreTokenGetter{
		Store: st,
	}
	ctx := context.Background()
	tok, err := g.GetToken(ctx)
	c.Assert(err, gc.Equals, nil)
	c.Assert(tok, jc.DeepEquals, testToken)
	st.CheckCalls(c, []testing.StubCall{{
		FuncName: "Get",
	}})
}

func (s *storeTokenGetterSuite) TestTokenNotInStore(c *gc.C) {
	testToken := &usso.SSOData{
		ConsumerKey:    "consumerkey",
		ConsumerSecret: "consumersecret",
		Realm:          "realm",
		TokenKey:       "tokenkey",
		TokenName:      "tokenname",
		TokenSecret:    "tokensecret",
	}
	st := &testTokenStore{}
	st.SetErrors(errgo.New("not found"))
	fg := &testTokenGetter{
		tok: testToken,
	}
	g := &ussologin.StoreTokenGetter{
		Store:       st,
		TokenGetter: fg,
	}
	ctx := context.Background()
	tok, err := g.GetToken(ctx)
	c.Assert(err, gc.Equals, nil)
	c.Assert(tok, jc.DeepEquals, testToken)
	st.CheckCalls(c, []testing.StubCall{{
		FuncName: "Get",
	}, {
		FuncName: "Put",
		Args:     []interface{}{testToken},
	}})
	fg.CheckCalls(c, []testing.StubCall{{
		FuncName: "GetToken",
		Args:     []interface{}{ctx},
	}})
}

type formTokenGetterSuite struct {
	testing.CleanupSuite
}

var _ = gc.Suite(&formTokenGetterSuite{})

func (s *formTokenGetterSuite) TestCorrectUserPasswordSentToUSSOServer(c *gc.C) {
	ussoStub := &ussoServerStub{}
	s.PatchValue(ussologin.Server, ussoStub)
	tg := ussologin.FormTokenGetter{
		Filler: &testFiller{
			map[string]interface{}{
				ussologin.UserKey: "foobar",
				ussologin.PassKey: "pass",
				ussologin.OTPKey:  "1234",
			}},
		Name: "testToken",
	}
	_, err := tg.GetToken(context.Background())
	c.Assert(err, gc.Equals, nil)
	ussoStub.CheckCall(c, 0, "GetTokenWithOTP", "foobar", "pass", "1234", "testToken")
}

func (s *formTokenGetterSuite) TestLoginFailsToGetToken(c *gc.C) {
	ussoStub := &ussoServerStub{}
	ussoStub.SetErrors(errgo.New("something failed"))
	s.PatchValue(ussologin.Server, ussoStub)
	tg := ussologin.FormTokenGetter{
		Filler: &testFiller{
			map[string]interface{}{
				ussologin.UserKey: "foobar",
				ussologin.PassKey: "pass",
				ussologin.OTPKey:  "1234",
			}},
		Name: "testToken",
	}
	_, err := tg.GetToken(context.Background())
	c.Assert(err, gc.ErrorMatches, "cannot get token: something failed")
}

func (s *formTokenGetterSuite) TestFailedToReadLoginParameters(c *gc.C) {
	ussoStub := &ussoServerStub{}
	s.PatchValue(ussologin.Server, ussoStub)
	tg := ussologin.FormTokenGetter{
		Filler: &errFiller{},
	}
	_, err := tg.GetToken(context.Background())
	c.Assert(err, gc.ErrorMatches, "cannot read login parameters: something failed")
	ussoStub.CheckNoCalls(c)
}

type testFiller struct {
	form map[string]interface{}
}

func (t *testFiller) Fill(f form.Form) (map[string]interface{}, error) {
	return t.form, nil
}

type errFiller struct{}

func (t *errFiller) Fill(f form.Form) (map[string]interface{}, error) {
	return nil, errgo.New("something failed")
}

type ussoServerStub struct {
	testing.Stub
}

func (u *ussoServerStub) GetTokenWithOTP(email, password, otp, tokenName string) (*usso.SSOData, error) {
	u.AddCall("GetTokenWithOTP", email, password, otp, tokenName)
	return &usso.SSOData{}, u.NextErr()
}

type testTokenGetter struct {
	testing.Stub
	tok *usso.SSOData
}

func (g *testTokenGetter) GetToken(ctx context.Context) (*usso.SSOData, error) {
	g.MethodCall(g, "GetToken", ctx)
	return g.tok, g.NextErr()
}

type testTokenStore struct {
	testing.Stub
	tok *usso.SSOData
}

func (m *testTokenStore) Put(tok *usso.SSOData) error {
	m.MethodCall(m, "Put", tok)
	m.tok = tok
	return m.NextErr()
}

func (m *testTokenStore) Get() (*usso.SSOData, error) {
	m.MethodCall(m, "Get")
	return m.tok, m.NextErr()
}
