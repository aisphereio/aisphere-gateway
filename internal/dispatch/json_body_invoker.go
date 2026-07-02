package dispatch

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"strconv"

	iamv1 "aisphere-iam/api/iam/v1"
	"github.com/aisphereio/kernel/errorx"
	"github.com/aisphereio/kernel/gatewayx"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type MessageFactory func() proto.Message

type JSONBodyInvoker struct {
	Next      gatewayx.UpstreamInvoker
	Requests  map[string]MessageFactory
	Unmarshal protojson.UnmarshalOptions
}

func NewJSONBodyInvoker(next gatewayx.UpstreamInvoker, requests map[string]MessageFactory) JSONBodyInvoker {
	return JSONBodyInvoker{
		Next:     next,
		Requests: requests,
		Unmarshal: protojson.UnmarshalOptions{
			DiscardUnknown: true,
		},
	}
}

func (i JSONBodyInvoker) Invoke(ctx context.Context, match gatewayx.RouteMatch, req gatewayx.DispatchRequest, target string) (gatewayx.DispatchResponse, error) {
	if i.Next == nil {
		return gatewayx.DispatchResponse{}, fmt.Errorf("gateway dispatch: nil upstream invoker")
	}
	if factory := i.Requests[match.Route.Upstream.Operation]; factory != nil {
		msg, err := i.decode(factory, req.Body, req.Headers["X-Gateway-Raw-Query"])
		if err != nil {
			return gatewayx.DispatchResponse{Status: 400}, err
		}
		if msg != nil {
			req.Body = msg
		}
	}
	return i.Next.Invoke(ctx, match, req, target)
}

func (i JSONBodyInvoker) decode(factory MessageFactory, body any, rawQuery string) (proto.Message, error) {
	raw, ok := bodyBytes(body)
	msg := factory()
	if msg == nil {
		return nil, fmt.Errorf("gateway dispatch: nil protobuf request")
	}
	hasQuery := populateQuery(msg, rawQuery)
	if !ok || len(bytes.TrimSpace(raw)) == 0 {
		if hasQuery {
			return msg, nil
		}
		return nil, nil
	}
	if err := i.Unmarshal.Unmarshal(raw, msg); err != nil {
		return nil, errorx.BadRequest("GATEWAY_INVALID_JSON_BODY", err.Error())
	}
	_ = populateQuery(msg, rawQuery)
	return msg, nil
}

func bodyBytes(body any) ([]byte, bool) {
	switch v := body.(type) {
	case nil:
		return nil, false
	case []byte:
		return v, true
	case string:
		return []byte(v), true
	default:
		return nil, false
	}
}

func populateQuery(msg proto.Message, rawQuery string) bool {
	if rawQuery == "" {
		return false
	}
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return false
	}
	switch m := msg.(type) {
	case *iamv1.BuildLoginURLRequest:
		m.RedirectUri = firstQuery(values, "redirect_uri", m.GetRedirectUri())
		m.State = firstQuery(values, "state", m.GetState())
		m.Scope = firstQuery(values, "scope", m.GetScope())
		m.OrgId = firstQuery(values, "org_id", m.GetOrgId())
		m.AppId = firstQuery(values, "app_id", m.GetAppId())
		return true
	case *iamv1.GetMeRequest:
		m.IncludeProfile = boolQuery(values, "include_profile", m.GetIncludeProfile())
		return true
	case *iamv1.ListUsersRequest:
		m.Query = firstQuery(values, "query", m.GetQuery())
		m.GroupId = firstQuery(values, "group_id", m.GetGroupId())
		m.Role = firstQuery(values, "role", m.GetRole())
		m.PageToken = firstQuery(values, "page_token", m.GetPageToken())
		m.PageSize = int32Query(values, "page_size", m.GetPageSize())
		return true
	case *iamv1.ListGroupsRequest:
		m.ParentId = firstQuery(values, "parent_id", m.GetParentId())
		m.Type = firstQuery(values, "type", m.GetType())
		m.UserId = firstQuery(values, "user_id", m.GetUserId())
		m.IncludeInherited = boolQuery(values, "include_inherited", m.GetIncludeInherited())
		return true
	default:
		return false
	}
}

func firstQuery(values url.Values, key, fallback string) string {
	if value := values.Get(key); value != "" {
		return value
	}
	return fallback
}

func boolQuery(values url.Values, key string, fallback bool) bool {
	value := values.Get(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func int32Query(values url.Values, key string, fallback int32) int32 {
	value := values.Get(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return fallback
	}
	return int32(parsed)
}
