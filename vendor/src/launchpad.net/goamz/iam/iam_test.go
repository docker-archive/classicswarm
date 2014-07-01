package iam_test

import (
	"launchpad.net/goamz/aws"
	"launchpad.net/goamz/iam"
	"launchpad.net/goamz/testutil"
	. "launchpad.net/gocheck"
	"strings"
	"testing"
)

func Test(t *testing.T) {
	TestingT(t)
}

type S struct {
	iam *iam.IAM
}

var _ = Suite(&S{})

var testServer = testutil.NewHTTPServer()

func (s *S) SetUpSuite(c *C) {
	testServer.Start()
	auth := aws.Auth{"abc", "123"}
	s.iam = iam.New(auth, aws.Region{IAMEndpoint: testServer.URL})
}

func (s *S) TearDownSuite(c *C) {
	testServer.Stop()
}

func (s *S) TearDownTest(c *C) {
	testServer.Flush()
}

func (s *S) TestCreateUser(c *C) {
	testServer.Response(200, nil, CreateUserExample)
	resp, err := s.iam.CreateUser("Bob", "/division_abc/subdivision_xyz/")
	values := testServer.WaitRequest().URL.Query()
	c.Assert(values.Get("Action"), Equals, "CreateUser")
	c.Assert(values.Get("UserName"), Equals, "Bob")
	c.Assert(values.Get("Path"), Equals, "/division_abc/subdivision_xyz/")
	c.Assert(err, IsNil)
	c.Assert(resp.RequestId, Equals, "7a62c49f-347e-4fc4-9331-6e8eEXAMPLE")
	expected := iam.User{
		Path: "/division_abc/subdivision_xyz/",
		Name: "Bob",
		Id:   "AIDACKCEVSQ6C2EXAMPLE",
		Arn:  "arn:aws:iam::123456789012:user/division_abc/subdivision_xyz/Bob",
	}
	c.Assert(resp.User, DeepEquals, expected)
}

func (s *S) TestCreateUserConflict(c *C) {
	testServer.Response(409, nil, DuplicateUserExample)
	resp, err := s.iam.CreateUser("Bob", "/division_abc/subdivision_xyz/")
	testServer.WaitRequest()
	c.Assert(resp, IsNil)
	c.Assert(err, NotNil)
	e, ok := err.(*iam.Error)
	c.Assert(ok, Equals, true)
	c.Assert(e.Message, Equals, "User with name Bob already exists.")
	c.Assert(e.Code, Equals, "EntityAlreadyExists")
}

func (s *S) TestGetUser(c *C) {
	testServer.Response(200, nil, GetUserExample)
	resp, err := s.iam.GetUser("Bob")
	values := testServer.WaitRequest().URL.Query()
	c.Assert(values.Get("Action"), Equals, "GetUser")
	c.Assert(values.Get("UserName"), Equals, "Bob")
	c.Assert(err, IsNil)
	c.Assert(resp.RequestId, Equals, "7a62c49f-347e-4fc4-9331-6e8eEXAMPLE")
	expected := iam.User{
		Path: "/division_abc/subdivision_xyz/",
		Name: "Bob",
		Id:   "AIDACKCEVSQ6C2EXAMPLE",
		Arn:  "arn:aws:iam::123456789012:user/division_abc/subdivision_xyz/Bob",
	}
	c.Assert(resp.User, DeepEquals, expected)
}

func (s *S) TestDeleteUser(c *C) {
	testServer.Response(200, nil, RequestIdExample)
	resp, err := s.iam.DeleteUser("Bob")
	values := testServer.WaitRequest().URL.Query()
	c.Assert(values.Get("Action"), Equals, "DeleteUser")
	c.Assert(values.Get("UserName"), Equals, "Bob")
	c.Assert(err, IsNil)
	c.Assert(resp.RequestId, Equals, "7a62c49f-347e-4fc4-9331-6e8eEXAMPLE")
}

func (s *S) TestCreateGroup(c *C) {
	testServer.Response(200, nil, CreateGroupExample)
	resp, err := s.iam.CreateGroup("Admins", "/admins/")
	values := testServer.WaitRequest().URL.Query()
	c.Assert(values.Get("Action"), Equals, "CreateGroup")
	c.Assert(values.Get("GroupName"), Equals, "Admins")
	c.Assert(values.Get("Path"), Equals, "/admins/")
	c.Assert(err, IsNil)
	c.Assert(resp.Group.Path, Equals, "/admins/")
	c.Assert(resp.Group.Name, Equals, "Admins")
	c.Assert(resp.Group.Id, Equals, "AGPACKCEVSQ6C2EXAMPLE")
	c.Assert(resp.RequestId, Equals, "7a62c49f-347e-4fc4-9331-6e8eEXAMPLE")
}

func (s *S) TestCreateGroupWithoutPath(c *C) {
	testServer.Response(200, nil, CreateGroupExample)
	_, err := s.iam.CreateGroup("Managers", "")
	values := testServer.WaitRequest().URL.Query()
	c.Assert(values.Get("Action"), Equals, "CreateGroup")
	c.Assert(err, IsNil)
	_, ok := map[string][]string(values)["Path"]
	c.Assert(ok, Equals, false)
}

func (s *S) TestDeleteGroup(c *C) {
	testServer.Response(200, nil, RequestIdExample)
	resp, err := s.iam.DeleteGroup("Admins")
	values := testServer.WaitRequest().URL.Query()
	c.Assert(values.Get("Action"), Equals, "DeleteGroup")
	c.Assert(values.Get("GroupName"), Equals, "Admins")
	c.Assert(err, IsNil)
	c.Assert(resp.RequestId, Equals, "7a62c49f-347e-4fc4-9331-6e8eEXAMPLE")
}

func (s *S) TestListGroups(c *C) {
	testServer.Response(200, nil, ListGroupsExample)
	resp, err := s.iam.Groups("/division_abc/")
	values := testServer.WaitRequest().URL.Query()
	c.Assert(values.Get("Action"), Equals, "ListGroups")
	c.Assert(values.Get("PathPrefix"), Equals, "/division_abc/")
	c.Assert(err, IsNil)
	c.Assert(resp.RequestId, Equals, "7a62c49f-347e-4fc4-9331-6e8eEXAMPLE")
	expected := []iam.Group{
		{
			Path: "/division_abc/subdivision_xyz/",
			Name: "Admins",
			Id:   "AGPACKCEVSQ6C2EXAMPLE",
			Arn:  "arn:aws:iam::123456789012:group/Admins",
		},
		{
			Path: "/division_abc/subdivision_xyz/product_1234/engineering/",
			Name: "Test",
			Id:   "AGP2MAB8DPLSRHEXAMPLE",
			Arn:  "arn:aws:iam::123456789012:group/division_abc/subdivision_xyz/product_1234/engineering/Test",
		},
		{
			Path: "/division_abc/subdivision_xyz/product_1234/",
			Name: "Managers",
			Id:   "AGPIODR4TAW7CSEXAMPLE",
			Arn:  "arn:aws:iam::123456789012:group/division_abc/subdivision_xyz/product_1234/Managers",
		},
	}
	c.Assert(resp.Groups, DeepEquals, expected)
}

func (s *S) TestListGroupsWithoutPathPrefix(c *C) {
	testServer.Response(200, nil, ListGroupsExample)
	_, err := s.iam.Groups("")
	values := testServer.WaitRequest().URL.Query()
	c.Assert(values.Get("Action"), Equals, "ListGroups")
	c.Assert(err, IsNil)
	_, ok := map[string][]string(values)["PathPrefix"]
	c.Assert(ok, Equals, false)
}

func (s *S) TestCreateAccessKey(c *C) {
	testServer.Response(200, nil, CreateAccessKeyExample)
	resp, err := s.iam.CreateAccessKey("Bob")
	values := testServer.WaitRequest().URL.Query()
	c.Assert(values.Get("Action"), Equals, "CreateAccessKey")
	c.Assert(values.Get("UserName"), Equals, "Bob")
	c.Assert(err, IsNil)
	c.Assert(resp.AccessKey.UserName, Equals, "Bob")
	c.Assert(resp.AccessKey.Id, Equals, "AKIAIOSFODNN7EXAMPLE")
	c.Assert(resp.AccessKey.Secret, Equals, "wJalrXUtnFEMI/K7MDENG/bPxRfiCYzEXAMPLEKEY")
	c.Assert(resp.AccessKey.Status, Equals, "Active")
}

func (s *S) TestDeleteAccessKey(c *C) {
	testServer.Response(200, nil, RequestIdExample)
	resp, err := s.iam.DeleteAccessKey("ysa8hasdhasdsi", "Bob")
	values := testServer.WaitRequest().URL.Query()
	c.Assert(values.Get("Action"), Equals, "DeleteAccessKey")
	c.Assert(values.Get("AccessKeyId"), Equals, "ysa8hasdhasdsi")
	c.Assert(values.Get("UserName"), Equals, "Bob")
	c.Assert(err, IsNil)
	c.Assert(resp.RequestId, Equals, "7a62c49f-347e-4fc4-9331-6e8eEXAMPLE")
}

func (s *S) TestDeleteAccessKeyBlankUserName(c *C) {
	testServer.Response(200, nil, RequestIdExample)
	_, err := s.iam.DeleteAccessKey("ysa8hasdhasdsi", "")
	c.Assert(err, IsNil)
	values := testServer.WaitRequest().URL.Query()
	c.Assert(values.Get("Action"), Equals, "DeleteAccessKey")
	c.Assert(values.Get("AccessKeyId"), Equals, "ysa8hasdhasdsi")
	_, ok := map[string][]string(values)["UserName"]
	c.Assert(ok, Equals, false)
}

func (s *S) TestAccessKeys(c *C) {
	testServer.Response(200, nil, ListAccessKeyExample)
	resp, err := s.iam.AccessKeys("Bob")
	values := testServer.WaitRequest().URL.Query()
	c.Assert(values.Get("Action"), Equals, "ListAccessKeys")
	c.Assert(values.Get("UserName"), Equals, "Bob")
	c.Assert(err, IsNil)
	c.Assert(resp.RequestId, Equals, "7a62c49f-347e-4fc4-9331-6e8eEXAMPLE")
	c.Assert(resp.AccessKeys, HasLen, 2)
	c.Assert(resp.AccessKeys[0].Id, Equals, "AKIAIOSFODNN7EXAMPLE")
	c.Assert(resp.AccessKeys[0].UserName, Equals, "Bob")
	c.Assert(resp.AccessKeys[0].Status, Equals, "Active")
	c.Assert(resp.AccessKeys[1].Id, Equals, "AKIAI44QH8DHBEXAMPLE")
	c.Assert(resp.AccessKeys[1].UserName, Equals, "Bob")
	c.Assert(resp.AccessKeys[1].Status, Equals, "Inactive")
}

func (s *S) TestAccessKeysBlankUserName(c *C) {
	testServer.Response(200, nil, ListAccessKeyExample)
	_, err := s.iam.AccessKeys("")
	c.Assert(err, IsNil)
	values := testServer.WaitRequest().URL.Query()
	c.Assert(values.Get("Action"), Equals, "ListAccessKeys")
	_, ok := map[string][]string(values)["UserName"]
	c.Assert(ok, Equals, false)
}

func (s *S) TestGetUserPolicy(c *C) {
	testServer.Response(200, nil, GetUserPolicyExample)
	resp, err := s.iam.GetUserPolicy("Bob", "AllAccessPolicy")
	values := testServer.WaitRequest().URL.Query()
	c.Assert(values.Get("Action"), Equals, "GetUserPolicy")
	c.Assert(values.Get("UserName"), Equals, "Bob")
	c.Assert(values.Get("PolicyName"), Equals, "AllAccessPolicy")
	c.Assert(err, IsNil)
	c.Assert(resp.Policy.UserName, Equals, "Bob")
	c.Assert(resp.Policy.Name, Equals, "AllAccessPolicy")
	c.Assert(strings.TrimSpace(resp.Policy.Document), Equals, `{"Statement":[{"Effect":"Allow","Action":"*","Resource":"*"}]}`)
	c.Assert(resp.RequestId, Equals, "7a62c49f-347e-4fc4-9331-6e8eEXAMPLE")
}

func (s *S) TestPutUserPolicy(c *C) {
	document := `{
		"Statement": [
		{
			"Action": [
				"s3:*"
			],
			"Effect": "Allow",
			"Resource": [
				"arn:aws:s3:::8shsns19s90ajahadsj/*",
				"arn:aws:s3:::8shsns19s90ajahadsj"
			]
		}]
	}`
	testServer.Response(200, nil, RequestIdExample)
	resp, err := s.iam.PutUserPolicy("Bob", "AllAccessPolicy", document)
	req := testServer.WaitRequest()
	c.Assert(req.Method, Equals, "POST")
	c.Assert(req.FormValue("Action"), Equals, "PutUserPolicy")
	c.Assert(req.FormValue("PolicyName"), Equals, "AllAccessPolicy")
	c.Assert(req.FormValue("UserName"), Equals, "Bob")
	c.Assert(req.FormValue("PolicyDocument"), Equals, document)
	c.Assert(req.FormValue("Version"), Equals, "2010-05-08")
	c.Assert(err, IsNil)
	c.Assert(resp.RequestId, Equals, "7a62c49f-347e-4fc4-9331-6e8eEXAMPLE")
}

func (s *S) TestDeleteUserPolicy(c *C) {
	testServer.Response(200, nil, RequestIdExample)
	resp, err := s.iam.DeleteUserPolicy("Bob", "AllAccessPolicy")
	values := testServer.WaitRequest().URL.Query()
	c.Assert(values.Get("Action"), Equals, "DeleteUserPolicy")
	c.Assert(values.Get("PolicyName"), Equals, "AllAccessPolicy")
	c.Assert(values.Get("UserName"), Equals, "Bob")
	c.Assert(err, IsNil)
	c.Assert(resp.RequestId, Equals, "7a62c49f-347e-4fc4-9331-6e8eEXAMPLE")
}
