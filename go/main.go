package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

type user struct {
	Name       string            `json:"name"`
	Email      string            `json:"email"`
	Attributes map[string]string `json:"attributes"`
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
				AttributeName: aws.String("Name"),
				AttributeType: aws.String("S"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("Name"),
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
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}

		tableName := "MyTable"
		item := map[string]*dynamodb.AttributeValue{}
		item["Name"] = &dynamodb.AttributeValue{S: aws.String(p.Name)}
		item["Email"] = &dynamodb.AttributeValue{S: aws.String(p.Email)}
		for k, v := range p.Attributes {
			item[k] = &dynamodb.AttributeValue{S: aws.String(v)}
		}

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

		for {
			input := &dynamodb.ScanInput{
				TableName: aws.String(tableName),
				Limit:     aws.Int64(int64(limit)),
			}
			if lastEvaluatedKey != nil {
				input.ExclusiveStartKey = lastEvaluatedKey
			}

			result, err := dynamoDBClient.Scan(input)
			if err != nil {
				fmt.Println(err)
				http.Error(w, "Failed to list items", http.StatusInternalServerError)
				return
			}

			for _, item := range result.Items {
				json.NewEncoder(w).Encode(item)
			}

			lastEvaluatedKey = result.LastEvaluatedKey
			if lastEvaluatedKey == nil {
				break
			}
		}
	})

	http.ListenAndServe(":8081", nil)
}
