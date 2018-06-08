// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package candidtest_test

import (
	jc "github.com/juju/testing/checkers"
	"golang.org/x/net/context"
	gc "gopkg.in/check.v1"
	"gopkg.in/macaroon-bakery.v2/bakery"
	"gopkg.in/macaroon-bakery.v2/bakery/identchecker"
	"gopkg.in/macaroon-bakery.v2/httpbakery"

	"gopkg.in/CanonicalLtd/candidclient.v1"
	"gopkg.in/CanonicalLtd/candidclient.v1/candidtest"
	candidparams "gopkg.in/CanonicalLtd/candidclient.v1/params"
)

type suite struct{}

var _ = gc.Suite(&suite{})

func (*suite) TestDischarge(c *gc.C) {
	ctx := context.TODO()
	srv := candidtest.NewServer()
	srv.AddUser("server-user", candidtest.GroupListGroup)
	srv.AddUser("bob", "somegroup")
	client := srv.Client("bob")

	key, err := bakery.GenerateKey()
	c.Assert(err, gc.IsNil)
	b := identchecker.NewBakery(identchecker.BakeryParams{
		Key:            key,
		Locator:        srv,
		IdentityClient: srv.CandidClient("server-user"),
	})
	m, err := b.Oven.NewMacaroon(
		ctx,
		bakery.LatestVersion,
		candidclient.IdentityCaveats(srv.URL.String()),
		identchecker.LoginOp,
	)
	c.Assert(err, gc.IsNil)

	ms, err := client.DischargeAll(ctx, m)
	c.Assert(err, gc.IsNil)

	// Make sure that the macaroon discharged correctly and that it
	// has the right declared caveats.
	authInfo, err := b.Checker.Auth(ms).Allow(ctx, identchecker.LoginOp)
	c.Assert(err, gc.IsNil)
	c.Assert(authInfo.Identity, gc.NotNil)
	ident := authInfo.Identity.(candidclient.Identity)
	c.Assert(ident.Id(), gc.Equals, "bob")
	username, err := ident.Username()
	c.Assert(err, gc.IsNil)
	c.Assert(username, gc.Equals, "bob")
	groups, err := ident.Groups()
	c.Assert(err, gc.IsNil)
	c.Assert(groups, jc.DeepEquals, []string{"somegroup"})
}

func (*suite) TestDischargeDefaultUser(c *gc.C) {
	ctx := context.TODO()
	srv := candidtest.NewServer()
	srv.SetDefaultUser("bob")

	key, err := bakery.GenerateKey()
	c.Assert(err, gc.IsNil)
	b := identchecker.NewBakery(identchecker.BakeryParams{
		Key:            key,
		Locator:        srv,
		IdentityClient: srv.CandidClient("server-user"),
	})
	m, err := b.Oven.NewMacaroon(
		ctx,
		bakery.LatestVersion,
		candidclient.IdentityCaveats(srv.URL.String()),
		identchecker.LoginOp,
	)
	c.Assert(err, gc.IsNil)

	client := httpbakery.NewClient()
	ms, err := client.DischargeAll(ctx, m)
	c.Assert(err, gc.IsNil)

	// Make sure that the macaroon discharged correctly and that it
	// has the right declared caveats.
	authInfo, err := b.Checker.Auth(ms).Allow(ctx, identchecker.LoginOp)
	c.Assert(err, gc.IsNil)
	c.Assert(authInfo.Identity, gc.NotNil)
	ident := authInfo.Identity.(candidclient.Identity)
	c.Assert(ident.Id(), gc.Equals, "bob")
	username, err := ident.Username()
	c.Assert(err, gc.IsNil)
	c.Assert(username, gc.Equals, "bob")
	groups, err := ident.Groups()
	c.Assert(err, gc.IsNil)
	c.Assert(groups, gc.HasLen, 0)
}

func (*suite) TestGroups(c *gc.C) {
	srv := candidtest.NewServer()
	srv.AddUser("server-user", candidtest.GroupListGroup)
	srv.AddUser("bob", "beatles", "bobbins")
	srv.AddUser("alice")

	client := srv.CandidClient("server-user")
	groups, err := client.UserGroups(context.TODO(), &candidparams.UserGroupsRequest{
		Username: "bob",
	})
	c.Assert(err, gc.IsNil)
	c.Assert(groups, jc.DeepEquals, []string{"beatles", "bobbins"})

	groups, err = client.UserGroups(context.TODO(), &candidparams.UserGroupsRequest{
		Username: "alice",
	})
	c.Assert(err, gc.IsNil)
	c.Assert(groups, gc.HasLen, 0)
}

func (s *suite) TestAddUserWithExistingGroups(c *gc.C) {
	srv := candidtest.NewServer()
	srv.AddUser("alice", "anteaters")
	srv.AddUser("alice")
	srv.AddUser("alice", "goof", "anteaters")

	client := srv.CandidClient("alice")
	groups, err := client.UserGroups(context.TODO(), &candidparams.UserGroupsRequest{
		Username: "alice",
	})
	c.Assert(err, gc.IsNil)
	c.Assert(groups, jc.DeepEquals, []string{"anteaters", "goof"})
}
