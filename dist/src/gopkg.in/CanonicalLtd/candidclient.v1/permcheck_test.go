// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package candidclient_test

import (
	"time"

	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"gopkg.in/CanonicalLtd/candidclient.v1"
	"gopkg.in/CanonicalLtd/candidclient.v1/candidtest"
)

type permCheckerSuite struct {
}

var _ = gc.Suite(&permCheckerSuite{})

func (s *permCheckerSuite) TestPermChecker(c *gc.C) {
	srv := candidtest.NewServer()
	srv.AddUser("server-user", candidtest.GroupListGroup)
	srv.AddUser("alice", "somegroup")

	client, err := candidclient.New(candidclient.NewParams{
		BaseURL: srv.URL.String(),
		Client:  srv.Client("server-user"),
	})
	c.Assert(err, gc.IsNil)

	pc := candidclient.NewPermChecker(client, time.Hour)

	// No permissions always yields false.
	ok, err := pc.Allow("bob", nil)
	c.Assert(err, gc.IsNil)
	c.Assert(ok, gc.Equals, false)

	// If the user isn't found, we return a (false, nil)
	ok, err = pc.Allow("bob", []string{"beatles"})
	c.Assert(err, gc.IsNil)
	c.Assert(ok, gc.Equals, false)

	// If the perms allow everyone, it's ok
	ok, err = pc.Allow("bob", []string{"noone", "everyone"})
	c.Assert(err, gc.IsNil)
	c.Assert(ok, gc.Equals, true)

	// If the perms allow everyone@somewhere, it's ok.
	ok, err = pc.Allow("bob@somewhere", []string{"everyone@somewhere"})
	c.Assert(err, gc.IsNil)
	c.Assert(ok, gc.Equals, true)

	// Check that the everyone@x logic works with multiple @s.
	ok, err = pc.Allow("bob@foo@somewhere@else", []string{"everyone@somewhere@else"})
	c.Assert(err, gc.IsNil)
	c.Assert(ok, gc.Equals, true)

	// Check that we're careful enough about "everyone" as a prefix
	// to a user name.
	ok, err = pc.Allow("bobx", []string{"everyonex"})
	c.Assert(err, gc.IsNil)
	c.Assert(ok, gc.Equals, false)

	// If the perms allow the user itself, it's ok
	ok, err = pc.Allow("bob", []string{"noone", "bob"})
	c.Assert(err, gc.IsNil)
	c.Assert(ok, gc.Equals, true)

	srv.AddUser("bob", "beatles")

	// The group details are currently cached by the client,
	// so the original request will still fail.
	ok, err = pc.Allow("bob", []string{"beatles"})
	c.Assert(err, gc.IsNil)
	c.Assert(ok, gc.Equals, false)

	// Clearing the cache allows it to succeed.
	pc.CacheEvictAll()
	ok, err = pc.Allow("bob", []string{"beatles"})
	c.Assert(err, gc.IsNil)
	c.Assert(ok, gc.Equals, true)
}

func (s *permCheckerSuite) TestGroupCache(c *gc.C) {
	srv := candidtest.NewServer()
	srv.AddUser("server-user", candidtest.GroupListGroup)
	srv.AddUser("alice", "somegroup", "othergroup")

	client, err := candidclient.New(candidclient.NewParams{
		BaseURL: srv.URL.String(),
		Client:  srv.Client("server-user"),
	})
	c.Assert(err, gc.IsNil)

	cache := candidclient.NewGroupCache(client, time.Hour)

	// If the user isn't found, we retturn no groups.
	g, err := cache.Groups("bob")
	c.Assert(err, gc.IsNil)
	c.Assert(g, gc.HasLen, 0)

	g, err = cache.Groups("alice")
	c.Assert(err, gc.IsNil)
	c.Assert(g, jc.DeepEquals, []string{"othergroup", "somegroup"})

	srv.AddUser("bob", "beatles")

	// The group details are currently cached by the client,
	// so we'll still see the original group membership.
	g, err = cache.Groups("bob")
	c.Assert(err, gc.IsNil)
	c.Assert(g, gc.HasLen, 0)

	// Clearing the cache allows it to succeed.
	cache.CacheEvictAll()
	g, err = cache.Groups("bob")
	c.Assert(err, gc.IsNil)
	c.Assert(g, jc.DeepEquals, []string{"beatles"})
}
