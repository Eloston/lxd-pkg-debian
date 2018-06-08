package candidclient_test

import (
	"sort"

	jc "github.com/juju/testing/checkers"
	"golang.org/x/net/context"
	gc "gopkg.in/check.v1"
	"gopkg.in/errgo.v1"
	"gopkg.in/macaroon-bakery.v2/bakery"
	"gopkg.in/macaroon-bakery.v2/bakery/identchecker"
	"gopkg.in/macaroon-bakery.v2/httpbakery"

	"gopkg.in/CanonicalLtd/candidclient.v1"
	"gopkg.in/CanonicalLtd/candidclient.v1/candidtest"
)

type clientSuite struct{}

var _ = gc.Suite(&clientSuite{})

func (*clientSuite) TestIdentityClient(c *gc.C) {
	srv := candidtest.NewServer()
	srv.AddUser("bob", "alice", "charlie")
	testIdentityClient(c,
		srv.CandidClient("bob"),
		srv.Client("bob"),
		"bob", "bob", []string{"alice", "charlie"},
	)
}

func (*clientSuite) TestIdentityClientWithDomainStrip(c *gc.C) {
	srv := candidtest.NewServer()
	srv.AddUser("bob@usso", "alice@usso", "charlie@elsewhere")
	testIdentityClient(c,
		candidclient.StripDomain(srv.CandidClient("bob@usso"), "usso"),
		srv.Client("bob@usso"),
		"bob@usso", "bob", []string{"alice", "charlie@elsewhere"},
	)
}

func (*clientSuite) TestIdentityClientWithDomainStripNoDomains(c *gc.C) {
	srv := candidtest.NewServer()
	srv.AddUser("bob", "alice", "charlie")
	testIdentityClient(c,
		candidclient.StripDomain(srv.CandidClient("bob"), "usso"),
		srv.Client("bob"),
		"bob", "bob", []string{"alice", "charlie"},
	)
}

// testIdentityClient tests that the given identity client can be used to
// create a third party caveat that when discharged provides
// an Identity with the given id, user name and groups.
func testIdentityClient(c *gc.C, candidClient identchecker.IdentityClient, bclient *httpbakery.Client, expectId, expectUser string, expectGroups []string) {
	kr := httpbakery.NewThirdPartyLocator(nil, nil)
	kr.AllowInsecure()
	b := identchecker.NewBakery(identchecker.BakeryParams{
		Locator:        kr,
		Key:            bakery.MustGenerateKey(),
		IdentityClient: candidClient,
	})
	_, authErr := b.Checker.Auth().Allow(context.TODO(), identchecker.LoginOp)
	derr := errgo.Cause(authErr).(*bakery.DischargeRequiredError)

	m, err := b.Oven.NewMacaroon(context.TODO(), bakery.LatestVersion, derr.Caveats, derr.Ops...)
	c.Assert(err, gc.IsNil)

	ms, err := bclient.DischargeAll(context.TODO(), m)
	c.Assert(err, gc.IsNil)

	// Make sure that the macaroon discharged correctly and that it
	// has the right declared caveats.
	authInfo, err := b.Checker.Auth(ms).Allow(context.TODO(), identchecker.LoginOp)
	c.Assert(err, gc.IsNil)

	c.Assert(authInfo.Identity, gc.NotNil)
	c.Assert(authInfo.Identity.Id(), gc.Equals, expectId)
	c.Assert(authInfo.Identity.Domain(), gc.Equals, "")

	user := authInfo.Identity.(candidclient.Identity)

	u, err := user.Username()
	c.Assert(err, gc.IsNil)
	c.Assert(u, gc.Equals, expectUser)
	ok, err := user.Allow(context.TODO(), []string{expectGroups[0]})
	c.Assert(err, gc.IsNil)
	c.Assert(ok, gc.Equals, true)

	groups, err := user.Groups()
	c.Assert(err, gc.IsNil)
	sort.Strings(groups)
	c.Assert(groups, jc.DeepEquals, expectGroups)
}
