//
// goamz - Go packages to interact with the Amazon Web Services.
//
//   https://wiki.ubuntu.com/goamz
//
// Copyright (c) 2011 Memeo Inc.
//
// Written by Prudhvi Krishna Surapaneni <me@prudhvi.net>

// This package is in an experimental state, and does not currently
// follow conventions and style of the rest of goamz or common
// Go conventions. It must be polished before it's considered a
// first-class package in goamz.
package sns

// BUG(niemeyer): Package needs significant clean up.

// BUG(niemeyer): Topic values in responses are not being initialized
// properly, since they're supposed to reference *SNS.

// BUG(niemeyer): Package needs documentation.

// BUG(niemeyer): Message.Message should be "Payload []byte"

// BUG(niemeyer): Message.SNS must be dropped.

import (
	"encoding/xml"
	"errors"
	"launchpad.net/goamz/aws"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// The SNS type encapsulates operation with an SNS region.
type SNS struct {
	aws.Auth
	aws.Region
	private byte // Reserve the right of using private data.
}

type Topic struct {
	SNS      *SNS
	TopicArn string
}

func New(auth aws.Auth, region aws.Region) *SNS {
	return &SNS{auth, region, 0}
}

type Message struct {
	SNS     *SNS
	Topic   *Topic
	Message [8192]byte
	Subject string
}

type Subscription struct {
	Endpoint        string
	Owner           string
	Protocol        string
	SubscriptionArn string
	TopicArn        string
}

func (topic *Topic) Message(message [8192]byte, subject string) *Message {
	return &Message{topic.SNS, topic, message, subject}
}

type ResponseMetadata struct {
	RequestId string  `xml:"ResponseMetadata>RequestId"`
	BoxUsage  float64 `xml:"ResponseMetadata>BoxUsage"`
}

type ListTopicsResp struct {
	Topics    []Topic `xml:"ListTopicsResult>Topics>member"`
	NextToken string
	ResponseMetadata
}

type CreateTopicResp struct {
	Topic Topic `xml:"CreateTopicResult"`
	ResponseMetadata
}

type DeleteTopicResp struct {
	ResponseMetadata
}

type ListSubscriptionsResp struct {
	Subscriptions []Subscription `xml:"ListSubscriptionsResult>Subscriptions>member"`
	NextToken     string
	ResponseMetadata
}

type AttributeEntry struct {
	Key   string `xml:"key"`
	Value string `xml:"value"`
}

type GetTopicAttributesResp struct {
	Attributes []AttributeEntry `xml:"GetTopicAttributesResult>Attributes>entry"`
	ResponseMetadata
}

func makeParams(action string) map[string]string {
	params := make(map[string]string)
	params["Action"] = action
	return params
}

// ListTopics
//
// See http://goo.gl/lfrMK for more details.
func (sns *SNS) ListTopics(NextToken *string) (resp *ListTopicsResp, err error) {
	resp = &ListTopicsResp{}
	params := makeParams("ListTopics")
	if NextToken != nil {
		params["NextToken"] = *NextToken
	}
	err = sns.query(nil, nil, params, resp)
	return
}

// CreateTopic
//
// See http://goo.gl/m9aAt for more details.
func (sns *SNS) CreateTopic(Name string) (resp *CreateTopicResp, err error) {
	resp = &CreateTopicResp{}
	params := makeParams("CreateTopic")
	params["Name"] = Name
	err = sns.query(nil, nil, params, resp)
	return
}

// DeleteTopic
//
// See http://goo.gl/OXNcY for more details.
func (sns *SNS) DeleteTopic(topic Topic) (resp *DeleteTopicResp, err error) {
	resp = &DeleteTopicResp{}
	params := makeParams("DeleteTopic")
	params["TopicArn"] = topic.TopicArn
	err = sns.query(nil, nil, params, resp)
	return
}

// Delete
//
// Helper function for deleting a topic
func (topic *Topic) Delete() (resp *DeleteTopicResp, err error) {
	return topic.SNS.DeleteTopic(*topic)
}

// ListSubscriptions
//
// See http://goo.gl/k3aGn for more details.
func (sns *SNS) ListSubscriptions(NextToken *string) (resp *ListSubscriptionsResp, err error) {
	resp = &ListSubscriptionsResp{}
	params := makeParams("ListSubscriptions")
	if NextToken != nil {
		params["NextToken"] = *NextToken
	}
	err = sns.query(nil, nil, params, resp)
	return
}

// GetTopicAttributes
//
// See http://goo.gl/WXRoX for more details.
func (sns *SNS) GetTopicAttributes(TopicArn string) (resp *GetTopicAttributesResp, err error) {
	resp = &GetTopicAttributesResp{}
	params := makeParams("GetTopicAttributes")
	params["TopicArn"] = TopicArn
	err = sns.query(nil, nil, params, resp)
	return
}

type PublishOpt struct {
	Message          string
	MessageStructure string
	Subject          string
	TopicArn         string
}

type PublishResp struct {
	MessageId string `xml:"PublishResult>MessageId"`
	ResponseMetadata
}

// Publish
//
// See http://goo.gl/AY2D8 for more details.
func (sns *SNS) Publish(options *PublishOpt) (resp *PublishResp, err error) {
	resp = &PublishResp{}
	params := makeParams("Publish")

	if options.Subject != "" {
		params["Subject"] = options.Subject
	}

	if options.MessageStructure != "" {
		params["MessageStructure"] = options.MessageStructure
	}

	if options.Message != "" {
		params["Message"] = options.Message
	}

	if options.TopicArn != "" {
		params["TopicArn"] = options.TopicArn
	}

	err = sns.query(nil, nil, params, resp)
	return
}

type SetTopicAttributesResponse struct {
	ResponseMetadata
}

// SetTopicAttributes
//
// See http://goo.gl/oVYW7 for more details.
func (sns *SNS) SetTopicAttributes(AttributeName, AttributeValue, TopicArn string) (resp *SetTopicAttributesResponse, err error) {
	resp = &SetTopicAttributesResponse{}
	params := makeParams("SetTopicAttributes")

	if AttributeName == "" || TopicArn == "" {
		return nil, errors.New("Invalid Attribute Name or TopicArn")
	}

	params["AttributeName"] = AttributeName
	params["AttributeValue"] = AttributeValue
	params["TopicArn"] = TopicArn

	err = sns.query(nil, nil, params, resp)
	return
}

type SubscribeResponse struct {
	SubscriptionArn string `xml:"SubscribeResult>SubscriptionArn"`
	ResponseMetadata
}

// Subscribe
//
// See http://goo.gl/c3iGS for more details.
func (sns *SNS) Subscribe(Endpoint, Protocol, TopicArn string) (resp *SubscribeResponse, err error) {
	resp = &SubscribeResponse{}
	params := makeParams("Subscribe")

	params["Endpoint"] = Endpoint
	params["Protocol"] = Protocol
	params["TopicArn"] = TopicArn

	err = sns.query(nil, nil, params, resp)
	return
}

type UnsubscribeResponse struct {
	ResponseMetadata
}

// Unsubscribe
//
// See http://goo.gl/4l5Ge for more details.
func (sns *SNS) Unsubscribe(SubscriptionArn string) (resp *UnsubscribeResponse, err error) {
	resp = &UnsubscribeResponse{}
	params := makeParams("Unsubscribe")

	params["SubscriptionArn"] = SubscriptionArn

	err = sns.query(nil, nil, params, resp)
	return
}

type ConfirmSubscriptionResponse struct {
	SubscriptionArn string `xml:"ConfirmSubscriptionResult>SubscriptionArn"`
	ResponseMetadata
}

type ConfirmSubscriptionOpt struct {
	AuthenticateOnUnsubscribe string
	Token                     string
	TopicArn                  string
}

// ConfirmSubscription
//
// See http://goo.gl/3hXzH for more details.
func (sns *SNS) ConfirmSubscription(options *ConfirmSubscriptionOpt) (resp *ConfirmSubscriptionResponse, err error) {
	resp = &ConfirmSubscriptionResponse{}
	params := makeParams("ConfirmSubscription")

	if options.AuthenticateOnUnsubscribe != "" {
		params["AuthenticateOnUnsubscribe"] = options.AuthenticateOnUnsubscribe
	}

	params["Token"] = options.Token
	params["TopicArn"] = options.TopicArn

	err = sns.query(nil, nil, params, resp)
	return
}

type Permission struct {
	ActionName string
	AccountId  string
}

type AddPermissionResponse struct {
	ResponseMetadata
}

// AddPermission
//
// See http://goo.gl/mbY4a for more details.
func (sns *SNS) AddPermission(permissions []Permission, Label, TopicArn string) (resp *AddPermissionResponse, err error) {
	resp = &AddPermissionResponse{}
	params := makeParams("AddPermission")

	for i, p := range permissions {
		params["AWSAccountId.member."+strconv.Itoa(i+1)] = p.AccountId
		params["ActionName.member."+strconv.Itoa(i+1)] = p.ActionName
	}

	params["Label"] = Label
	params["TopicArn"] = TopicArn

	err = sns.query(nil, nil, params, resp)
	return
}

type RemovePermissionResponse struct {
	ResponseMetadata
}

// RemovePermission
//
// See http://goo.gl/wGl5j for more details.
func (sns *SNS) RemovePermission(Label, TopicArn string) (resp *RemovePermissionResponse, err error) {
	resp = &RemovePermissionResponse{}
	params := makeParams("RemovePermission")

	params["Label"] = Label
	params["TopicArn"] = TopicArn

	err = sns.query(nil, nil, params, resp)
	return
}

type ListSubscriptionByTopicResponse struct {
	Subscriptions []Subscription `xml:"ListSubscriptionsByTopicResult>Subscriptions>member"`
	ResponseMetadata
}

type ListSubscriptionByTopicOpt struct {
	NextToken string
	TopicArn  string
}

// ListSubscriptionByTopic
//
// See http://goo.gl/LaVcC for more details.
func (sns *SNS) ListSubscriptionByTopic(options *ListSubscriptionByTopicOpt) (resp *ListSubscriptionByTopicResponse, err error) {
	resp = &ListSubscriptionByTopicResponse{}
	params := makeParams("ListSbubscriptionByTopic")

	if options.NextToken != "" {
		params["NextToken"] = options.NextToken
	}

	params["TopicArn"] = options.TopicArn

	err = sns.query(nil, nil, params, resp)
	return
}

type Error struct {
	StatusCode int
	Code       string
	Message    string
	RequestId  string
}

func (err *Error) Error() string {
	return err.Message
}

type xmlErrors struct {
	RequestId string
	Errors    []Error `xml:"Errors>Error"`
}

func (sns *SNS) query(topic *Topic, message *Message, params map[string]string, resp interface{}) error {
	params["Timestamp"] = time.Now().UTC().Format(time.RFC3339)
	u, err := url.Parse(sns.Region.SNSEndpoint)
	if err != nil {
		return err
	}

	sign(sns.Auth, "GET", "/", params, u.Host)
	u.RawQuery = multimap(params).Encode()
	r, err := http.Get(u.String())
	if err != nil {
		return err
	}
	defer r.Body.Close()

	//dump, _ := http.DumpResponse(r, true)
	//println("DUMP:\n", string(dump))
	//return nil

	if r.StatusCode != 200 {
		return buildError(r)
	}
	err = xml.NewDecoder(r.Body).Decode(resp)
	return err
}

func buildError(r *http.Response) error {
	errors := xmlErrors{}
	xml.NewDecoder(r.Body).Decode(&errors)
	var err Error
	if len(errors.Errors) > 0 {
		err = errors.Errors[0]
	}
	err.RequestId = errors.RequestId
	err.StatusCode = r.StatusCode
	if err.Message == "" {
		err.Message = r.Status
	}
	return &err
}

func multimap(p map[string]string) url.Values {
	q := make(url.Values, len(p))
	for k, v := range p {
		q[k] = []string{v}
	}
	return q
}
