package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/google/uuid"
)

type user struct {
	Name       string            `json:"name"`
	Email      string            `json:"email"`
	Attributes map[string]string `json:"attributes"`
}

type dbUser struct {
	Id         string            `json:"id"`
	Name       string            `json:"name"`
	Email      string            `json:"email"`
	Attributes map[string]string `json:"attributes"`
}

type listResponse struct {
	Items             []dbUser `json:"items"`
	ContinuationToken string   `json:"continuationToken"`
}

func main() {
	sess, err := session.NewSession(
		&aws.Config{
			Region:      aws.String("local"),
			Endpoint:    aws.String("http://localhost:8000"),
			Credentials: credentials.NewStaticCredentials("AKID", "SECRET_KEY", "TOKEN"),
		},
		nil)

	if err != nil {
		fmt.Println(err)
		return
	}

	dynamoDBClient := dynamodb.New(sess)

	_, err = dynamoDBClient.DeleteTable(&dynamodb.DeleteTableInput{
		TableName: aws.String("MyTable"),
	})

	if err != nil {
		fmt.Println(err)
	}

	_, err = dynamoDBClient.CreateTable(&dynamodb.CreateTableInput{
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("Id"),
				AttributeType: aws.String("S"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("Id"),
				KeyType:       aws.String("HASH"),
			},
		},
		TableName: aws.String("MyTable"),
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(5000),
			WriteCapacityUnits: aws.Int64(5000),
		},
	})

	if err != nil {
		fmt.Println(err)
		return
	}

	http.HandleFunc("/create", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Invalid request method", http.StatusBadRequest)
			return
		}

		var p user
		err := json.NewDecoder(r.Body).Decode(&p)
		if err != nil {
			fmt.Println(err)
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}

		tableName := "MyTable"
		item := map[string]*dynamodb.AttributeValue{}
		// generate random uuid
		uuid := uuid.New().String()
		item["Id"] = &dynamodb.AttributeValue{S: aws.String(uuid)}
		item["Name"] = &dynamodb.AttributeValue{S: aws.String(p.Name)}
		item["Email"] = &dynamodb.AttributeValue{S: aws.String(p.Email)}

		attributesMap := make(map[string]*dynamodb.AttributeValue)
		for k, v := range p.Attributes {
			attributesMap[k] = &dynamodb.AttributeValue{S: aws.String(v)}
		}
		item["Attributes"] = &dynamodb.AttributeValue{M: attributesMap}

		input := &dynamodb.PutItemInput{
			TableName: aws.String(tableName),
			Item:      item,
		}
		_, err = dynamoDBClient.PutItem(input)
		if err != nil {
			http.Error(w, "Failed to store item", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	})

	http.HandleFunc("/list", func(w http.ResponseWriter, r *http.Request) {
		tableName := "MyTable"
		limit := 1000
		var lastEvaluatedKey map[string]*dynamodb.AttributeValue

		input := &dynamodb.ScanInput{
			TableName: aws.String(tableName),
			Limit:     aws.Int64(int64(limit)),
		}
		// Get the continuation token from the query parameter
		continuationToken := r.URL.Query().Get("continuationToken")
		if continuationToken != "" {
			input.ExclusiveStartKey = map[string]*dynamodb.AttributeValue{
				"Id": {
					S: aws.String(continuationToken),
				},
			}
		}
		result, err := dynamoDBClient.Scan(input)
		if err != nil {
			fmt.Println(err)
			http.Error(w, "Failed to list items", http.StatusInternalServerError)
			return
		}

		listResp := listResponse{}
		for _, item := range result.Items {
			// convert dynamodb item to dbUser struct
			attributes := map[string]string{}
			for k, v := range item["Attributes"].M {
				attributes[k] = *v.S
			}

			user := dbUser{
				Id:         *item["Id"].S,
				Name:       *item["Name"].S,
				Email:      *item["Email"].S,
				Attributes: attributes,
			}

			listResp.Items = append(listResp.Items, user)
		}

		lastEvaluatedKey = result.LastEvaluatedKey
		if lastEvaluatedKey != nil {
			// Generate the continuation token for the next iteration
			continuationToken = *lastEvaluatedKey["Id"].S
			listResp.ContinuationToken = continuationToken
		}

		json.NewEncoder(w).Encode(listResp)
	})

	http.ListenAndServe(":8081", nil)
}
