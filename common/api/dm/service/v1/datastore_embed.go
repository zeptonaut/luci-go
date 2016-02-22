// Copyright 2015 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dm

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/luci/gae/service/datastore"
)

const flipMask uint32 = 0xFFFFFFFF

var _ datastore.PropertyConverter = (*Attempt_ID)(nil)
var _ datastore.PropertyConverter = (*Execution_ID)(nil)

// NewQuestID is a shorthand to New a new *Quest_ID
func NewQuestID(qst string) *Quest_ID {
	return &Quest_ID{qst}
}

// NewAttemptID is a shorthand to New a new *Attempt_ID
func NewAttemptID(qst string, aid uint32) *Attempt_ID {
	return &Attempt_ID{qst, aid}
}

// NewExecutionID is a shorthand to New a new *Execution_ID
func NewExecutionID(qst string, aid, eid uint32) *Execution_ID {
	return &Execution_ID{qst, aid, eid}
}

// ToProperty implements datastore.PropertyConverter for the purpose of
// embedding this Attempt_ID as the ID of a luci/gae compatible datastore
// object. The numerical id field is stored as an inverted, hex-encoded string,
// so that Attempt_ID{"quest", 1} would encode as "quest|fffffffe". This is done
// so that the __key__ ordering in the dm application prefers to order the most
// recent attempts first.
//
// The go representation will always have the normal non-flipped numerical id.
func (a *Attempt_ID) ToProperty() (datastore.Property, error) {
	return datastore.MkPropertyNI(a.DMEncoded()), nil
}

// FromProperty implements datastore.PropertyConverter
func (a *Attempt_ID) FromProperty(p datastore.Property) error {
	if p.Type() != datastore.PTString {
		return fmt.Errorf("wrong type for property: %s", p.Type())
	}
	return a.SetDMEncoded(p.Value().(string))
}

// DMEncoded returns the encoded string id for this Attempt. Numeric values are
// inverted if flip is true.
func (a *Attempt_ID) DMEncoded() string {
	return fmt.Sprintf("%s|%08x", a.Quest, flipMask^a.Id)
}

// SetDMEncoded decodes val into this Attempt_ID, returning an error if
// there's a problem. Numeric values are inverted if flip is true.
func (a *Attempt_ID) SetDMEncoded(val string) error {
	toks := strings.SplitN(val, "|", 2)
	if len(toks) != 2 {
		return fmt.Errorf("unable to parse Attempt id: %q", val)
	}
	an, err := strconv.ParseUint(toks[1], 16, 32)
	if err != nil {
		return err
	}

	a.Quest = toks[0]
	a.Id = flipMask ^ uint32(an)
	return nil
}

// ToProperty implements datastore.PropertyConverter for the purpose of
// embedding this Execution_ID as the ID of a luci/gae compatible datastore
// object. The numerical id field is stored as an inverted, hex-encoded string,
// so that Execution_ID{"quest", 1, 2} would encode as "quest|fffffffe|fffffffd".
// This is done so that the __key__ ordering in the dm application prefers to
// order the most recent executions first.
//
// The go representation will always have the normal non-flipped numerical ids.
func (e *Execution_ID) ToProperty() (datastore.Property, error) {
	return datastore.MkPropertyNI(e.DMEncoded()), nil
}

// FromProperty implements datastore.PropertyConverter
func (e *Execution_ID) FromProperty(p datastore.Property) error {
	if p.Type() != datastore.PTString {
		return fmt.Errorf("wrong type for property: %s", p.Type())
	}
	return e.SetDMEncoded(p.Value().(string))
}

// DMEncoded returns the encoded inverted string id for this Execution. Numeric
// values are inverted if flip is true.
func (e *Execution_ID) DMEncoded() string {
	return fmt.Sprintf("%s|%08x|%08x", e.Quest, flipMask^e.Attempt, flipMask^e.Id)
}

// SetDMEncoded decodes val into this Execution_ID, returning an error if
// there's a problem. Numeric values are inverted if flip is true.
func (e *Execution_ID) SetDMEncoded(val string) error {
	toks := strings.SplitN(val, "|", 3)
	if len(toks) != 3 {
		return fmt.Errorf("unable to parse Execution id: %q", val)
	}
	an, err := strconv.ParseUint(toks[1], 16, 32)
	if err != nil {
		return err
	}
	en, err := strconv.ParseUint(toks[2], 16, 32)
	if err != nil {
		return err
	}

	e.Quest = toks[0]
	e.Attempt = flipMask ^ uint32(an)
	e.Id = flipMask ^ uint32(en)
	return nil
}