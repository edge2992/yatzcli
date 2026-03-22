package lambda

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockDynamoDB struct {
	mu    sync.Mutex
	items []map[string]types.AttributeValue

	putCalls    int
	deleteCalls int
	scanCalls   int
}

func (m *mockDynamoDB) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.putCalls++
	m.items = append(m.items, params.Item)
	return &dynamodb.PutItemOutput{}, nil
}

func (m *mockDynamoDB) DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deleteCalls++
	playerID := params.Key["PlayerID"].(*types.AttributeValueMemberS).Value
	filtered := m.items[:0]
	for _, item := range m.items {
		id := item["PlayerID"].(*types.AttributeValueMemberS).Value
		if id != playerID {
			filtered = append(filtered, item)
		}
	}
	m.items = filtered
	return &dynamodb.DeleteItemOutput{}, nil
}

func (m *mockDynamoDB) Scan(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scanCalls++

	selfID := ""
	if v, ok := params.ExpressionAttributeValues[":self"]; ok {
		selfID = v.(*types.AttributeValueMemberS).Value
	}

	var result []map[string]types.AttributeValue
	for _, item := range m.items {
		id := item["PlayerID"].(*types.AttributeValueMemberS).Value
		if id != selfID {
			result = append(result, item)
			if params.Limit != nil && int32(len(result)) >= *params.Limit {
				break
			}
		}
	}
	return &dynamodb.ScanOutput{Items: result, Count: int32(len(result))}, nil
}

type mockAPIGateway struct {
	mu           sync.Mutex
	sentMessages map[string][]byte
}

func newMockAPIGateway() *mockAPIGateway {
	return &mockAPIGateway{sentMessages: make(map[string][]byte)}
}

func (m *mockAPIGateway) PostToConnection(ctx context.Context, params *apigatewaymanagementapi.PostToConnectionInput, optFns ...func(*apigatewaymanagementapi.Options)) (*apigatewaymanagementapi.PostToConnectionOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentMessages[*params.ConnectionId] = params.Data
	return &apigatewaymanagementapi.PostToConnectionOutput{}, nil
}

func makeEvent(routeKey, connectionID, sourceIP, body string) events.APIGatewayWebsocketProxyRequest {
	return events.APIGatewayWebsocketProxyRequest{
		RequestContext: events.APIGatewayWebsocketProxyRequestContext{
			RouteKey:     routeKey,
			ConnectionID: connectionID,
			Identity: events.APIGatewayRequestIdentity{
				SourceIP: sourceIP,
			},
		},
		Body: body,
	}
}

func TestHandler_Connect(t *testing.T) {
	db := &mockDynamoDB{}
	apiGW := newMockAPIGateway()
	h := NewHandler(db, apiGW, "test-table")

	event := makeEvent("$connect", "conn-1", "1.2.3.4", "")
	resp, err := h.HandleRequest(context.Background(), event)

	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, 0, db.putCalls)
	assert.Equal(t, 0, db.deleteCalls)
}

func TestHandler_Message_FirstPlayer(t *testing.T) {
	db := &mockDynamoDB{}
	apiGW := newMockAPIGateway()
	h := NewHandler(db, apiGW, "test-table")

	body := `{"name":"Alice","port":8080}`
	event := makeEvent("$default", "conn-1", "1.2.3.4", body)
	resp, err := h.HandleRequest(context.Background(), event)

	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, 1, db.scanCalls)
	assert.Equal(t, 1, db.putCalls)
	assert.Equal(t, 0, len(apiGW.sentMessages))

	require.Len(t, db.items, 1)
	item := db.items[0]
	assert.Equal(t, "conn-1", item["PlayerID"].(*types.AttributeValueMemberS).Value)
	assert.Equal(t, "Alice", item["Name"].(*types.AttributeValueMemberS).Value)
	assert.Equal(t, "1.2.3.4:8080", item["Endpoint"].(*types.AttributeValueMemberS).Value)
}

func TestHandler_Message_SecondPlayer(t *testing.T) {
	db := &mockDynamoDB{}
	apiGW := newMockAPIGateway()
	h := NewHandler(db, apiGW, "test-table")

	// First player registers
	body1 := `{"name":"Alice","port":8080}`
	event1 := makeEvent("$default", "conn-1", "1.2.3.4", body1)
	_, err := h.HandleRequest(context.Background(), event1)
	require.NoError(t, err)

	// Second player matches
	body2 := `{"name":"Bob","port":9090}`
	event2 := makeEvent("$default", "conn-2", "5.6.7.8", body2)
	resp, err := h.HandleRequest(context.Background(), event2)

	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Both players should be notified
	require.Len(t, apiGW.sentMessages, 2)

	// Verify message to conn-2 (current / guest)
	var resultForGuest MatchResult
	require.NoError(t, json.Unmarshal(apiGW.sentMessages["conn-2"], &resultForGuest))
	assert.Equal(t, "1.2.3.4:8080", resultForGuest.OpponentAddr)
	assert.Equal(t, "Alice", resultForGuest.OpponentName)
	assert.False(t, resultForGuest.IsHost)

	// Verify message to conn-1 (waiting / host)
	var resultForHost MatchResult
	require.NoError(t, json.Unmarshal(apiGW.sentMessages["conn-1"], &resultForHost))
	assert.Equal(t, "5.6.7.8:9090", resultForHost.OpponentAddr)
	assert.Equal(t, "Bob", resultForHost.OpponentName)
	assert.True(t, resultForHost.IsHost)

	// Waiting table should be empty after match
	assert.Empty(t, db.items)
}

func TestHandler_Disconnect(t *testing.T) {
	db := &mockDynamoDB{}
	apiGW := newMockAPIGateway()
	h := NewHandler(db, apiGW, "test-table")

	// Register a player first
	body := `{"name":"Alice","port":8080}`
	event := makeEvent("$default", "conn-1", "1.2.3.4", body)
	_, err := h.HandleRequest(context.Background(), event)
	require.NoError(t, err)
	require.Len(t, db.items, 1)

	// Disconnect
	disconnectEvent := makeEvent("$disconnect", "conn-1", "1.2.3.4", "")
	resp, err := h.HandleRequest(context.Background(), disconnectEvent)

	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Empty(t, db.items)
}
