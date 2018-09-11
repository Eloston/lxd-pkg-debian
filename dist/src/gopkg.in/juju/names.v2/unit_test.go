// Copyright 2013 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package names_test

import (
	"fmt"

	gc "gopkg.in/check.v1"

	"gopkg.in/juju/names.v2"
)

type unitSuite struct{}

var _ = gc.Suite(&unitSuite{})

func (s *unitSuite) TestUnitTag(c *gc.C) {
	c.Assert(names.NewUnitTag("wordpress/2").String(), gc.Equals, "unit-wordpress-2")
	c.Assert(names.NewUnitTag("foo-bar/2").String(), gc.Equals, "unit-foo-bar-2")
}

var unitNameTests = []struct {
	pattern     string
	valid       bool
	application string
}{
	{pattern: "wordpress/42", valid: true, application: "wordpress"},
	{pattern: "rabbitmq-server/123", valid: true, application: "rabbitmq-server"},
	{pattern: "foo", valid: false},
	{pattern: "foo/", valid: false},
	{pattern: "bar/foo", valid: false},
	{pattern: "20/20", valid: false},
	{pattern: "foo-55", valid: false},
	{pattern: "foo-bar/123", valid: true, application: "foo-bar"},
	{pattern: "foo-bar/123/", valid: false},
	{pattern: "foo-bar/123-not", valid: false},
}

func (s *unitSuite) TestUnitNameFormats(c *gc.C) {
	for i, test := range unitNameTests {
		c.Logf("test %d: %q", i, test.pattern)
		c.Assert(names.IsValidUnit(test.pattern), gc.Equals, test.valid)
	}
}

func (s *unitSuite) TestInvalidUnitTagFormats(c *gc.C) {
	for i, test := range unitNameTests {
		if !test.valid {
			c.Logf("test %d: %q", i, test.pattern)
			expect := fmt.Sprintf("%q is not a valid unit name", test.pattern)
			testUnitTag := func() { names.NewUnitTag(test.pattern) }
			c.Assert(testUnitTag, gc.PanicMatches, expect)
		}
	}
}

func (s *unitSuite) TestNumber(c *gc.C) {
	u := names.UnitTag{}
	c.Assert(u.Number(), gc.Equals, 0)
	u = names.NewUnitTag("foo-t4/5")
	c.Assert(u.Number(), gc.Equals, 5)
}

func (s *applicationSuite) TestUnitApplication(c *gc.C) {
	for i, test := range unitNameTests {
		c.Logf("test %d: %q", i, test.pattern)
		if !test.valid {
			expect := fmt.Sprintf("%q is not a valid unit name", test.pattern)
			_, err := names.UnitApplication(test.pattern)
			c.Assert(err, gc.ErrorMatches, expect)
		} else {
			result, err := names.UnitApplication(test.pattern)
			c.Assert(err, gc.IsNil)
			c.Assert(result, gc.Equals, test.application)
		}
	}
}

var parseUnitTagTests = []struct {
	tag      string
	expected names.Tag
	err      error
}{{
	tag: "",
	err: names.InvalidTagError("", ""),
}, {
	tag:      "unit-dave-0",
	expected: names.NewUnitTag("dave/0"),
}, {
	tag:      "unit-foo-bar-0",
	expected: names.NewUnitTag("foo-bar/0"),
}, {
	tag: "dave",
	err: names.InvalidTagError("dave", ""),
}, {
	tag: "unit-dave",
	err: names.InvalidTagError("unit-dave", names.UnitTagKind), // not a valid unit name either
}, {
	tag: "application-dave",
	err: names.InvalidTagError("application-dave", names.UnitTagKind),
}}

func (s *unitSuite) TestParseUnitTag(c *gc.C) {
	for i, t := range parseUnitTagTests {
		c.Logf("test %d: %s", i, t.tag)
		got, err := names.ParseUnitTag(t.tag)
		if err != nil || t.err != nil {
			c.Check(err, gc.DeepEquals, t.err)
			continue
		}
		c.Check(got, gc.FitsTypeOf, t.expected)
		c.Check(got, gc.Equals, t.expected)
	}
}

var unitTagShortenedStringTests = []struct {
	unitName     string
	maxLength    int
	errorOutput  string
	expectedName string
}{{
	unitName:    "foobar/0",
	maxLength:   15,
	errorOutput: "max length must be at least 21, not 15",
}, {
	unitName:     "foobar/0",
	maxLength:    30,
	expectedName: "unit-foobar-0",
}, {
	unitName:     "veryveryverylonglongone/0",
	maxLength:    25,
	expectedName: "unit-veryverf4fa59c6-0",
}, {
	unitName:     "veryveryverylonglongone/20",
	maxLength:    25,
	expectedName: "unit-veryverf4fa59c6-20",
}, {
	unitName:     "veryveryverylonglongone/300",
	maxLength:    25,
	expectedName: "unit-veryverf4fa59c6-300",
}, {
	unitName:     "veryveryverylonglongone/4000",
	maxLength:    25,
	expectedName: "unit-veryverf4fa59c6-4000",
}, {
	unitName:     "veryveryverylonglongone/50000",
	maxLength:    25,
	expectedName: "unit-veryvef4fa59c6-50000",
}}

func (s *unitSuite) TestUnitTagShortenedString(c *gc.C) {
	for i, t := range unitTagShortenedStringTests {
		c.Logf("test %d: %s %d", i, t.unitName, t.maxLength)
		tag := names.NewUnitTag(t.unitName)
		tagString, err := tag.ShortenedString(t.maxLength)
		if t.errorOutput != "" {
			c.Check(tagString, gc.Equals, "")
			c.Check(err, gc.ErrorMatches, t.errorOutput)
		} else {
			c.Check(tagString, gc.Equals, t.expectedName)
			c.Check(err, gc.IsNil)
		}
	}
}
