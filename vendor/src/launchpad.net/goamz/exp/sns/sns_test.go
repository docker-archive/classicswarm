package sns_test

import (
	"launchpad.net/goamz/aws"
	"launchpad.net/goamz/exp/sns"
	"launchpad.net/goamz/testutil"
	. "launchpad.net/gocheck"
	"testing"
)

func Test(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&S{})

type S struct {
	sns *sns.SNS
}

var testServer = testutil.NewHTTPServer()

func (s *S) SetUpSuite(c *C) {
	testServer.Start()
	auth := aws.Auth{"abc", "123"}
	s.sns = sns.New(auth, aws.Region{SNSEndpoint: testServer.URL})
}

func (s *S) TearDownSuite(c *C) {
	testServer.Stop()
}

func (s *S) TearDownTest(c *C) {
	testServer.Flush()
}

func (s *S) TestListTopicsOK(c *C) {
	testServer.Response(200, nil, TestListTopicsXmlOK)

	resp, err := s.sns.ListTopics(nil)
	req := testServer.WaitRequest()

	c.Assert(req.Method, Equals, "GET")
	c.Assert(req.URL.Path, Equals, "/")
	c.Assert(req.Header["Date"], Not(Equals), "")

	c.Assert(resp.ResponseMetadata.RequestId, Equals, "bd10b26c-e30e-11e0-ba29-93c3aca2f103")
	c.Assert(err, IsNil)
}

func (s *S) TestCreateTopic(c *C) {
	testServer.Response(200, nil, TestCreateTopicXmlOK)

	resp, err := s.sns.CreateTopic("My-Topic")
	req := testServer.WaitRequest()

	c.Assert(req.Method, Equals, "GET")
	c.Assert(req.URL.Path, Equals, "/")
	c.Assert(req.Header["Date"], Not(Equals), "")

	c.Assert(resp.Topic.TopicArn, Equals, "arn:aws:sns:us-east-1:123456789012:My-Topic")
	c.Assert(resp.ResponseMetadata.RequestId, Equals, "a8dec8b3-33a4-11df-8963-01868b7c937a")
	c.Assert(err, IsNil)
}

func (s *S) TestDeleteTopic(c *C) {
	testServer.Response(200, nil, TestDeleteTopicXmlOK)

	t := sns.Topic{nil, "arn:aws:sns:us-east-1:123456789012:My-Topic"}
	resp, err := s.sns.DeleteTopic(t)
	req := testServer.WaitRequest()

	c.Assert(req.Method, Equals, "GET")
	c.Assert(req.URL.Path, Equals, "/")
	c.Assert(req.Header["Date"], Not(Equals), "")

	c.Assert(resp.ResponseMetadata.RequestId, Equals, "f3aa9ac9-3c3d-11df-8235-9dab105e9c32")
	c.Assert(err, IsNil)
}

func (s *S) TestListSubscriptions(c *C) {
	testServer.Response(200, nil, TestListSubscriptionsXmlOK)

	resp, err := s.sns.ListSubscriptions(nil)
	req := testServer.WaitRequest()

	c.Assert(req.Method, Equals, "GET")
	c.Assert(req.URL.Path, Equals, "/")
	c.Assert(req.Header["Date"], Not(Equals), "")

	c.Assert(len(resp.Subscriptions), Not(Equals), 0)
	c.Assert(resp.Subscriptions[0].Protocol, Equals, "email")
	c.Assert(resp.Subscriptions[0].Endpoint, Equals, "example@amazon.com")
	c.Assert(resp.Subscriptions[0].SubscriptionArn, Equals, "arn:aws:sns:us-east-1:123456789012:My-Topic:80289ba6-0fd4-4079-afb4-ce8c8260f0ca")
	c.Assert(resp.Subscriptions[0].TopicArn, Equals, "arn:aws:sns:us-east-1:698519295917:My-Topic")
	c.Assert(resp.Subscriptions[0].Owner, Equals, "123456789012")
	c.Assert(err, IsNil)
}

func (s *S) TestGetTopicAttributes(c *C) {
	testServer.Response(200, nil, TestGetTopicAttributesXmlOK)

	resp, err := s.sns.GetTopicAttributes("arn:aws:sns:us-east-1:123456789012:My-Topic")
	req := testServer.WaitRequest()

	c.Assert(req.Method, Equals, "GET")
	c.Assert(req.URL.Path, Equals, "/")
	c.Assert(req.Header["Date"], Not(Equals), "")

	c.Assert(len(resp.Attributes), Not(Equals), 0)
	c.Assert(resp.Attributes[0].Key, Equals, "Owner")
	c.Assert(resp.Attributes[0].Value, Equals, "123456789012")
	c.Assert(resp.Attributes[1].Key, Equals, "Policy")
	c.Assert(resp.Attributes[1].Value, Equals, `{"Version":"2008-10-17","Id":"us-east-1/698519295917/test__default_policy_ID","Statement" : [{"Effect":"Allow","Sid":"us-east-1/698519295917/test__default_statement_ID","Principal" : {"AWS": "*"},"Action":["SNS:GetTopicAttributes","SNS:SetTopicAttributes","SNS:AddPermission","SNS:RemovePermission","SNS:DeleteTopic","SNS:Subscribe","SNS:ListSubscriptionsByTopic","SNS:Publish","SNS:Receive"],"Resource":"arn:aws:sns:us-east-1:698519295917:test","Condition" : {"StringLike" : {"AWS:SourceArn": "arn:aws:*:*:698519295917:*"}}}]}`)
	c.Assert(resp.ResponseMetadata.RequestId, Equals, "057f074c-33a7-11df-9540-99d0768312d3")
	c.Assert(err, IsNil)
}

func (s *S) TestPublish(c *C) {
	testServer.Response(200, nil, TestPublishXmlOK)

	pubOpt := &sns.PublishOpt{"foobar", "", "subject", "arn:aws:sns:us-east-1:123456789012:My-Topic"}
	resp, err := s.sns.Publish(pubOpt)
	req := testServer.WaitRequest()

	c.Assert(req.Method, Equals, "GET")
	c.Assert(req.URL.Path, Equals, "/")
	c.Assert(req.Header["Date"], Not(Equals), "")

	c.Assert(resp.MessageId, Equals, "94f20ce6-13c5-43a0-9a9e-ca52d816e90b")
	c.Assert(resp.ResponseMetadata.RequestId, Equals, "f187a3c1-376f-11df-8963-01868b7c937a")
	c.Assert(err, IsNil)
}

func (s *S) TestSetTopicAttributes(c *C) {
	testServer.Response(200, nil, TestSetTopicAttributesXmlOK)

	resp, err := s.sns.SetTopicAttributes("DisplayName", "MyTopicName", "arn:aws:sns:us-east-1:123456789012:My-Topic")
	req := testServer.WaitRequest()

	c.Assert(req.Method, Equals, "GET")
	c.Assert(req.URL.Path, Equals, "/")
	c.Assert(req.Header["Date"], Not(Equals), "")

	c.Assert(resp.ResponseMetadata.RequestId, Equals, "a8763b99-33a7-11df-a9b7-05d48da6f042")
	c.Assert(err, IsNil)
}

func (s *S) TestSubscribe(c *C) {
	testServer.Response(200, nil, TestSubscribeXmlOK)

	resp, err := s.sns.Subscribe("example@amazon.com", "email", "arn:aws:sns:us-east-1:123456789012:My-Topic")
	req := testServer.WaitRequest()

	c.Assert(req.Method, Equals, "GET")
	c.Assert(req.URL.Path, Equals, "/")
	c.Assert(req.Header["Date"], Not(Equals), "")

	c.Assert(resp.SubscriptionArn, Equals, "pending confirmation")
	c.Assert(resp.ResponseMetadata.RequestId, Equals, "a169c740-3766-11df-8963-01868b7c937a")
	c.Assert(err, IsNil)
}

func (s *S) TestUnsubscribe(c *C) {
	testServer.Response(200, nil, TestUnsubscribeXmlOK)

	resp, err := s.sns.Unsubscribe("arn:aws:sns:us-east-1:123456789012:My-Topic:a169c740-3766-11df-8963-01868b7c937a")
	req := testServer.WaitRequest()

	c.Assert(req.Method, Equals, "GET")
	c.Assert(req.URL.Path, Equals, "/")
	c.Assert(req.Header["Date"], Not(Equals), "")

	c.Assert(resp.ResponseMetadata.RequestId, Equals, "18e0ac39-3776-11df-84c0-b93cc1666b84")
	c.Assert(err, IsNil)
}

func (s *S) TestConfirmSubscription(c *C) {
	testServer.Response(200, nil, TestConfirmSubscriptionXmlOK)

	opt := &sns.ConfirmSubscriptionOpt{"", "51b2ff3edb475b7d91550e0ab6edf0c1de2a34e6ebaf6", "arn:aws:sns:us-east-1:123456789012:My-Topic"}
	resp, err := s.sns.ConfirmSubscription(opt)
	req := testServer.WaitRequest()

	c.Assert(req.Method, Equals, "GET")
	c.Assert(req.URL.Path, Equals, "/")
	c.Assert(req.Header["Date"], Not(Equals), "")

	c.Assert(resp.SubscriptionArn, Equals, "arn:aws:sns:us-east-1:123456789012:My-Topic:80289ba6-0fd4-4079-afb4-ce8c8260f0ca")
	c.Assert(resp.ResponseMetadata.RequestId, Equals, "7a50221f-3774-11df-a9b7-05d48da6f042")
	c.Assert(err, IsNil)
}

func (s *S) TestAddPermission(c *C) {
	testServer.Response(200, nil, TestAddPermissionXmlOK)
	perm := make([]sns.Permission, 2)
	perm[0].ActionName = "Publish"
	perm[1].ActionName = "GetTopicAttributes"
	perm[0].AccountId = "987654321000"
	perm[1].AccountId = "876543210000"

	resp, err := s.sns.AddPermission(perm, "NewPermission", "arn:aws:sns:us-east-1:123456789012:My-Topic")
	req := testServer.WaitRequest()

	c.Assert(req.Method, Equals, "GET")
	c.Assert(req.URL.Path, Equals, "/")
	c.Assert(req.Header["Date"], Not(Equals), "")

	c.Assert(resp.RequestId, Equals, "6a213e4e-33a8-11df-9540-99d0768312d3")
	c.Assert(err, IsNil)
}

func (s *S) TestRemovePermission(c *C) {
	testServer.Response(200, nil, TestRemovePermissionXmlOK)

	resp, err := s.sns.RemovePermission("NewPermission", "arn:aws:sns:us-east-1:123456789012:My-Topic")
	req := testServer.WaitRequest()

	c.Assert(req.Method, Equals, "GET")
	c.Assert(req.URL.Path, Equals, "/")
	c.Assert(req.Header["Date"], Not(Equals), "")

	c.Assert(resp.RequestId, Equals, "d170b150-33a8-11df-995a-2d6fbe836cc1")
	c.Assert(err, IsNil)
}

func (s *S) TestListSubscriptionByTopic(c *C) {
	testServer.Response(200, nil, TestListSubscriptionsByTopicXmlOK)

	opt := &sns.ListSubscriptionByTopicOpt{"", "arn:aws:sns:us-east-1:123456789012:My-Topic"}
	resp, err := s.sns.ListSubscriptionByTopic(opt)
	req := testServer.WaitRequest()

	c.Assert(req.Method, Equals, "GET")
	c.Assert(req.URL.Path, Equals, "/")
	c.Assert(req.Header["Date"], Not(Equals), "")

	c.Assert(len(resp.Subscriptions), Not(Equals), 0)
	c.Assert(resp.Subscriptions[0].TopicArn, Equals, "arn:aws:sns:us-east-1:123456789012:My-Topic")
	c.Assert(resp.Subscriptions[0].SubscriptionArn, Equals, "arn:aws:sns:us-east-1:123456789012:My-Topic:80289ba6-0fd4-4079-afb4-ce8c8260f0ca")
	c.Assert(resp.Subscriptions[0].Owner, Equals, "123456789012")
	c.Assert(resp.Subscriptions[0].Endpoint, Equals, "example@amazon.com")
	c.Assert(resp.Subscriptions[0].Protocol, Equals, "email")
	c.Assert(err, IsNil)
}
