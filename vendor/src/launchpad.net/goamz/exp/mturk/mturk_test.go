package mturk_test

import (
	"launchpad.net/goamz/aws"
	"launchpad.net/goamz/exp/mturk"
	"launchpad.net/goamz/testutil"
	. "launchpad.net/gocheck"
	"net/url"
	"testing"
)

func Test(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&S{})

type S struct {
	mturk *mturk.MTurk
}

var testServer = testutil.NewHTTPServer()

func (s *S) SetUpSuite(c *C) {
	testServer.Start()
	auth := aws.Auth{"abc", "123"}
	u, err := url.Parse(testServer.URL)
	if err != nil {
		panic(err.Error())
	}

	s.mturk = &mturk.MTurk{
		Auth: auth,
		URL:  u,
	}
}

func (s *S) TearDownSuite(c *C) {
	testServer.Stop()
}

func (s *S) TearDownTest(c *C) {
	testServer.Flush()
}

func (s *S) TestCreateHIT(c *C) {
	testServer.Response(200, nil, BasicHitResponse)

	question := mturk.ExternalQuestion{
		ExternalURL: "http://www.amazon.com",
		FrameHeight: 200,
	}
	reward := mturk.Price{
		Amount:       "0.01",
		CurrencyCode: "USD",
	}
	hit, err := s.mturk.CreateHIT("title", "description", question, reward, 1, 2, "key1,key2", 3, nil, "annotation")

	testServer.WaitRequest()

	c.Assert(err, IsNil)
	c.Assert(hit, NotNil)

	c.Assert(hit.HITId, Equals, "28J4IXKO2L927XKJTHO34OCDNASCDW")
	c.Assert(hit.HITTypeId, Equals, "2XZ7D1X3V0FKQVW7LU51S7PKKGFKDF")
}

func (s *S) TestSearchHITs(c *C) {
	testServer.Response(200, nil, SearchHITResponse)

	hitResult, err := s.mturk.SearchHITs()

	c.Assert(err, IsNil)
	c.Assert(hitResult, NotNil)

	c.Assert(hitResult.NumResults, Equals, uint(1))
	c.Assert(hitResult.PageNumber, Equals, uint(1))
	c.Assert(hitResult.TotalNumResults, Equals, uint(1))

	c.Assert(len(hitResult.HITs), Equals, 1)
	c.Assert(hitResult.HITs[0].HITId, Equals, "2BU26DG67D1XTE823B3OQ2JF2XWF83")
	c.Assert(hitResult.HITs[0].HITTypeId, Equals, "22OWJ5OPB0YV6IGL5727KP9U38P5XR")
	c.Assert(hitResult.HITs[0].CreationTime, Equals, "2011-12-28T19:56:20Z")
	c.Assert(hitResult.HITs[0].Title, Equals, "test hit")
	c.Assert(hitResult.HITs[0].Description, Equals, "please disregard, testing only")
	c.Assert(hitResult.HITs[0].HITStatus, Equals, "Reviewable")
	c.Assert(hitResult.HITs[0].MaxAssignments, Equals, uint(1))
	c.Assert(hitResult.HITs[0].Reward.Amount, Equals, "0.01")
	c.Assert(hitResult.HITs[0].Reward.CurrencyCode, Equals, "USD")
	c.Assert(hitResult.HITs[0].AutoApprovalDelayInSeconds, Equals, uint(2592000))
	c.Assert(hitResult.HITs[0].AssignmentDurationInSeconds, Equals, uint(30))
	c.Assert(hitResult.HITs[0].NumberOfAssignmentsPending, Equals, uint(0))
	c.Assert(hitResult.HITs[0].NumberOfAssignmentsAvailable, Equals, uint(1))
	c.Assert(hitResult.HITs[0].NumberOfAssignmentsCompleted, Equals, uint(0))
}
