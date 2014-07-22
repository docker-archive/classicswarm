//
// goamz - Go packages to interact with the Amazon Web Services.
//
//   https://wiki.ubuntu.com/goamz
//
// Copyright (c) 2011 Canonical Ltd.
//
// Written by Graham Miller <graham.miller@gmail.com>

// This package is in an experimental state, and does not currently
// follow conventions and style of the rest of goamz or common
// Go conventions. It must be polished before it's considered a
// first-class package in goamz.
package mturk

import (
	"encoding/xml"
	"errors"
	"fmt"
	"launchpad.net/goamz/aws"
	"net/http"
	//"net/http/httputil"
	"net/url"
	"strconv"
	"time"
)

type MTurk struct {
	aws.Auth
	URL *url.URL
}

func New(auth aws.Auth) *MTurk {
	mt := &MTurk{Auth: auth}
	var err error
	mt.URL, err = url.Parse("http://mechanicalturk.amazonaws.com/")
	if err != nil {
		panic(err.Error())
	}
	return mt
}

// ----------------------------------------------------------------------------
// Request dispatching logic.

// Error encapsulates an error returned by MTurk.
type Error struct {
	StatusCode int    // HTTP status code (200, 403, ...)
	Code       string // EC2 error code ("UnsupportedOperation", ...)
	Message    string // The human-oriented error message
	RequestId  string
}

func (err *Error) Error() string {
	return err.Message
}

// The request stanza included in several response types, for example
// in a "CreateHITResponse".  http://goo.gl/qGeKf
type xmlRequest struct {
	RequestId string
	IsValid   string
	Errors    []Error `xml:"Errors>Error"`
}

// Common price structure used in requests and responses
// http://goo.gl/tE4AV
type Price struct {
	Amount         string
	CurrencyCode   string
	FormattedPrice string
}

// Really just a country string
// http://goo.gl/mU4uG
type Locale string

// Data structure used to specify requirements for the worker
// used in CreateHIT, for example
// http://goo.gl/LvRo9
type QualificationRequirement struct {
	QualificationTypeId string
	Comparator          string
	IntegerValue        int
	LocaleValue         Locale
	RequiredToPreview   string
}

// Data structure holding the contents of an "external"
// question. http://goo.gl/NP8Aa
type ExternalQuestion struct {
	XMLName     xml.Name `xml:"http://mechanicalturk.amazonaws.com/AWSMechanicalTurkDataSchemas/2006-07-14/ExternalQuestion.xsd ExternalQuestion"`
	ExternalURL string
	FrameHeight int
}

// The data structure representing a "human interface task" (HIT)
// Currently only supports "external" questions, because Go
// structs don't support union types.  http://goo.gl/NP8Aa
// This type is returned, for example, from SearchHITs
// http://goo.gl/PskcX
type HIT struct {
	Request xmlRequest

	HITId                        string
	HITTypeId                    string
	CreationTime                 string
	Title                        string
	Description                  string
	Keywords                     string
	HITStatus                    string
	Reward                       Price
	LifetimeInSeconds            uint
	AssignmentDurationInSeconds  uint
	MaxAssignments               uint
	AutoApprovalDelayInSeconds   uint
	QualificationRequirement     QualificationRequirement
	Question                     ExternalQuestion
	RequesterAnnotation          string
	NumberofSimilarHITs          uint
	HITReviewStatus              string
	NumberOfAssignmentsPending   uint
	NumberOfAssignmentsAvailable uint
	NumberOfAssignmentsCompleted uint
}

// The main data structure returned by SearchHITs
// http://goo.gl/PskcX
type SearchHITsResult struct {
	NumResults      uint
	PageNumber      uint
	TotalNumResults uint
	HITs            []HIT `xml:"HIT"`
}

// The wrapper data structure returned by SearchHITs
// http://goo.gl/PskcX
type SearchHITsResponse struct {
	RequestId        string `xml:"OperationRequest>RequestId"`
	SearchHITsResult SearchHITsResult
}

// The wrapper data structure returned by CreateHIT
// http://goo.gl/PskcX
type CreateHITResponse struct {
	RequestId string `xml:"OperationRequest>RequestId"`
	HIT       HIT
}

// Corresponds to the "CreateHIT" operation of the Mechanical Turk
// API.  http://goo.gl/cDBRc Currently only supports "external"
// questions (see "HIT" struct above).  If "keywords", "maxAssignments",
// "qualificationRequirement" or "requesterAnnotation" are the zero
// value for their types, they will not be included in the request.
func (mt *MTurk) CreateHIT(title, description string, question ExternalQuestion, reward Price, assignmentDurationInSeconds, lifetimeInSeconds uint, keywords string, maxAssignments uint, qualificationRequirement *QualificationRequirement, requesterAnnotation string) (h *HIT, err error) {
	params := make(map[string]string)
	params["Title"] = title
	params["Description"] = description
	params["Question"], err = xmlEncode(&question)
	if err != nil {
		return
	}
	params["Reward.1.Amount"] = reward.Amount
	params["Reward.1.CurrencyCode"] = reward.CurrencyCode
	params["AssignmentDurationInSeconds"] = strconv.FormatUint(uint64(assignmentDurationInSeconds), 10)

	params["LifetimeInSeconds"] = strconv.FormatUint(uint64(lifetimeInSeconds), 10)
	if keywords != "" {
		params["Keywords"] = keywords
	}
	if maxAssignments != 0 {
		params["MaxAssignments"] = strconv.FormatUint(uint64(maxAssignments), 10)
	}
	if qualificationRequirement != nil {
		params["QualificationRequirement"], err = xmlEncode(qualificationRequirement)
		if err != nil {
			return
		}
	}
	if requesterAnnotation != "" {
		params["RequesterAnnotation"] = requesterAnnotation
	}

	var response CreateHITResponse
	err = mt.query(params, "CreateHIT", &response)
	if err == nil {
		h = &response.HIT
	}
	return
}

// Corresponds to the "CreateHIT" operation of the Mechanical Turk
// API, using an existing "hit type".  http://goo.gl/cDBRc Currently only
// supports "external" questions (see "HIT" struct above).  If
// "maxAssignments" or "requesterAnnotation" are the zero value for
// their types, they will not be included in the request.
func (mt *MTurk) CreateHITOfType(hitTypeId string, q ExternalQuestion, lifetimeInSeconds uint, maxAssignments uint, requesterAnnotation string) (h *HIT, err error) {
	params := make(map[string]string)
	params["HITTypeId"] = hitTypeId
	params["Question"], err = xmlEncode(&q)
	if err != nil {
		return
	}
	params["LifetimeInSeconds"] = strconv.FormatUint(uint64(lifetimeInSeconds), 10)
	if maxAssignments != 0 {
		params["MaxAssignments"] = strconv.FormatUint(uint64(maxAssignments), 10)
	}
	if requesterAnnotation != "" {
		params["RequesterAnnotation"] = requesterAnnotation
	}

	var response CreateHITResponse
	err = mt.query(params, "CreateHIT", &response)
	if err == nil {
		h = &response.HIT
	}
	return
}

// Corresponds to "SearchHITs" operation of Mechanical Turk. http://goo.gl/PskcX
// Currenlty supports none of the optional parameters.
func (mt *MTurk) SearchHITs() (s *SearchHITsResult, err error) {
	params := make(map[string]string)
	var response SearchHITsResponse
	err = mt.query(params, "SearchHITs", &response)
	if err == nil {
		s = &response.SearchHITsResult
	}
	return
}

// Adds common parameters to the "params" map, signs the request,
// adds the signature to the "params" map and sends the request
// to the server.  It then unmarshals the response in to the "resp"
// parameter using xml.Unmarshal()
func (mt *MTurk) query(params map[string]string, operation string, resp interface{}) error {
	service := "AWSMechanicalTurkRequester"
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05Z")

	params["AWSAccessKeyId"] = mt.Auth.AccessKey
	params["Service"] = service
	params["Timestamp"] = timestamp
	params["Operation"] = operation

	// make a copy
	url := *mt.URL

	sign(mt.Auth, service, operation, timestamp, params)
	url.RawQuery = multimap(params).Encode()
	r, err := http.Get(url.String())
	if err != nil {
		return err
	}
	//dump, _ := httputil.DumpResponse(r, true)
	//println("DUMP:\n", string(dump))
	if r.StatusCode != 200 {
		return errors.New(fmt.Sprintf("%d: unexpected status code", r.StatusCode))
	}
	dec := xml.NewDecoder(r.Body)
	err = dec.Decode(resp)
	r.Body.Close()
	return err
}

func multimap(p map[string]string) url.Values {
	q := make(url.Values, len(p))
	for k, v := range p {
		q[k] = []string{v}
	}
	return q
}

func xmlEncode(i interface{}) (s string, err error) {
	var buf []byte
	buf, err = xml.Marshal(i)
	if err != nil {
		return
	}
	s = string(buf)
	return
}
