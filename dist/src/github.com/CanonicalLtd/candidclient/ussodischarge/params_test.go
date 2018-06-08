// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package ussodischarge_test

import (
	"encoding/json"

	gc "gopkg.in/check.v1"

	"gopkg.in/CanonicalLtd/candidclient.v1/ussodischarge"
)

type paramsSuite struct {
}

var _ = gc.Suite(&paramsSuite{})

func (s *paramsSuite) TestUnmarshalUSSOMacaroon(c *gc.C) {
	data := []byte(`"MDAxYmxvY2F0aW9uIHRlc3QgbG9jYXRpb24KMDAxZGlkZW50aWZpZXIgdGVzdCBtYWNhcm9vbgowMDJmc2lnbmF0dXJlICaaplwsJeHwPuBK6er_d3DnEnSJ2b85-V9SXsiL6xWOCg"`)
	var m ussodischarge.USSOMacaroon
	err := json.Unmarshal(data, &m)
	c.Assert(err, gc.Equals, nil)
	c.Assert(string(m.Macaroon.Id()), gc.Equals, "test macaroon")
}

func (s *paramsSuite) TestUnmarshalUSSOMacaroonNotJSONString(c *gc.C) {
	data := []byte(`123`)
	var m ussodischarge.USSOMacaroon
	err := json.Unmarshal(data, &m)
	c.Assert(err, gc.ErrorMatches, `cannot unmarshal macaroon: json: cannot unmarshal number into Go value of type string`)
}

func (s *paramsSuite) TestUnmarshalUSSOMacaroonBadBase64(c *gc.C) {
	data := []byte(`"MDAxYmxvY2F0aW9uIHRlc3QgbG9jYXRpb24KMDAxZGlkZW50aWZpZXIgdGVzdCBtYWNhcm9vbgowMDJmc2lnbmF0dXJlICaaplwsJeHwPuBK6er/d3DnEnSJ2b85+V9SXsiL6xWOCg"`)
	var m ussodischarge.USSOMacaroon
	err := json.Unmarshal(data, &m)
	c.Assert(err, gc.ErrorMatches, `cannot unmarshal macaroon: illegal base64 data at input byte 111`)
}

func (s *paramsSuite) TestUnmarshalUSSOMacaroonBadBinary(c *gc.C) {
	data := []byte(`"NDAxYmxvY2F0aW9uIHRlc3QgbG9jYXRpb24KMDAxZGlkZW50aWZpZXIgdGVzdCBtYWNhcm9vbgowMDJmc2lnbmF0dXJlICaaplwsJeHwPuBK6er_d3DnEnSJ2b85-V9SXsiL6xWOCg"`)
	var m ussodischarge.USSOMacaroon
	err := json.Unmarshal(data, &m)
	c.Assert(err, gc.ErrorMatches, `cannot unmarshal macaroon: unmarshal v1: packet size too big`)
}
