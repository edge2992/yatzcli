package lambda

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const defaultTableName = "YatzcliWaitingPlayers"
const ttlDuration = 5 * time.Minute

// ClientMessage is what the client sends after connecting.
type ClientMessage struct {
	Name string `json:"name"`
	Port int    `json:"port"`
}

// MatchResult is sent to both matched players.
type MatchResult struct {
	OpponentAddr string `json:"opponent_addr"`
	OpponentName string `json:"opponent_name"`
	IsHost       bool   `json:"is_host"`
}

// DynamoDBClient interface for testing.
type DynamoDBClient interface {
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	Scan(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error)
}

// APIGatewayClient interface for testing.
type APIGatewayClient interface {
	PostToConnection(ctx context.Context, params *apigatewaymanagementapi.PostToConnectionInput, optFns ...func(*apigatewaymanagementapi.Options)) (*apigatewaymanagementapi.PostToConnectionOutput, error)
}

// Handler processes API Gateway WebSocket events.
type Handler struct {
	db    DynamoDBClient
	apiGW APIGatewayClient
	table string
}

// NewHandler creates a new Handler with the given clients and table name.
func NewHandler(db DynamoDBClient, apiGW APIGatewayClient, table string) *Handler {
	return &Handler{
		db:    db,
		apiGW: apiGW,
		table: table,
	}
}

// NewDefaultHandler creates a Handler using real AWS clients from environment.
func NewDefaultHandler(ctx context.Context) (*Handler, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	table := os.Getenv("TABLE_NAME")
	if table == "" {
		table = defaultTableName
	}

	endpoint := os.Getenv("WEBSOCKET_API_ENDPOINT")

	db := dynamodb.NewFromConfig(cfg)
	apiGW := apigatewaymanagementapi.NewFromConfig(cfg, func(o *apigatewaymanagementapi.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
	})

	return NewHandler(db, apiGW, table), nil
}

// HandleRequest routes the WebSocket event to the appropriate handler.
func (h *Handler) HandleRequest(ctx context.Context, event events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	routeKey := event.RequestContext.RouteKey

	var err error
	switch routeKey {
	case "$connect":
		err = h.handleConnect(ctx, event)
	case "$disconnect":
		err = h.handleDisconnect(ctx, event.RequestContext.ConnectionID)
	default:
		err = h.handleMessage(ctx, event)
	}

	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500, Body: err.Error()}, nil
	}
	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func (h *Handler) handleConnect(ctx context.Context, event events.APIGatewayWebsocketProxyRequest) error {
	return nil
}

func (h *Handler) handleDisconnect(ctx context.Context, connectionID string) error {
	_, err := h.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(h.table),
		Key: map[string]types.AttributeValue{
			"PlayerID": &types.AttributeValueMemberS{Value: connectionID},
		},
	})
	return err
}

func (h *Handler) handleMessage(ctx context.Context, event events.APIGatewayWebsocketProxyRequest) error {
	var msg ClientMessage
	if err := json.Unmarshal([]byte(event.Body), &msg); err != nil {
		return fmt.Errorf("parsing client message: %w", err)
	}

	connectionID := event.RequestContext.ConnectionID
	sourceIP := event.RequestContext.Identity.SourceIP
	endpoint := sourceIP + ":" + strconv.Itoa(msg.Port)

	// Scan for a waiting player (exclude self).
	// Note: do not use Limit with FilterExpression — DynamoDB applies
	// Limit before filtering, which can return 0 results even when
	// matching items exist.
	scanOut, err := h.db.Scan(ctx, &dynamodb.ScanInput{
		TableName:        aws.String(h.table),
		FilterExpression: aws.String("PlayerID <> :self"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":self": &types.AttributeValueMemberS{Value: connectionID},
		},
	})
	if err != nil {
		return fmt.Errorf("scanning waiting players: %w", err)
	}

	if len(scanOut.Items) > 0 {
		opponent := scanOut.Items[0]
		opponentID := opponent["PlayerID"].(*types.AttributeValueMemberS).Value
		opponentName := opponent["Name"].(*types.AttributeValueMemberS).Value
		opponentEndpoint := opponent["Endpoint"].(*types.AttributeValueMemberS).Value

		// Notify current player: you are the guest
		resultForCurrent := MatchResult{
			OpponentAddr: opponentEndpoint,
			OpponentName: opponentName,
			IsHost:       false,
		}
		if err := h.notifyPlayer(ctx, connectionID, resultForCurrent); err != nil {
			return fmt.Errorf("notifying current player: %w", err)
		}

		// Notify waiting player: you are the host
		resultForWaiting := MatchResult{
			OpponentAddr: endpoint,
			OpponentName: msg.Name,
			IsHost:       true,
		}
		if err := h.notifyPlayer(ctx, opponentID, resultForWaiting); err != nil {
			return fmt.Errorf("notifying waiting player: %w", err)
		}

		// Remove the matched opponent from the table
		_, err = h.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
			TableName: aws.String(h.table),
			Key: map[string]types.AttributeValue{
				"PlayerID": &types.AttributeValueMemberS{Value: opponentID},
			},
		})
		if err != nil {
			return fmt.Errorf("deleting matched player: %w", err)
		}

		return nil
	}

	// No waiting player found; register self
	now := time.Now()
	ttl := now.Add(ttlDuration).Unix()
	_, err = h.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(h.table),
		Item: map[string]types.AttributeValue{
			"PlayerID":  &types.AttributeValueMemberS{Value: connectionID},
			"Name":      &types.AttributeValueMemberS{Value: msg.Name},
			"Endpoint":  &types.AttributeValueMemberS{Value: endpoint},
			"CreatedAt": &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
			"TTL":       &types.AttributeValueMemberN{Value: strconv.FormatInt(ttl, 10)},
		},
	})
	if err != nil {
		return fmt.Errorf("registering player: %w", err)
	}

	return nil
}

func (h *Handler) notifyPlayer(ctx context.Context, connectionID string, result MatchResult) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshalling match result: %w", err)
	}

	_, err = h.apiGW.PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
		ConnectionId: aws.String(connectionID),
		Data:         data,
	})
	return err
}
