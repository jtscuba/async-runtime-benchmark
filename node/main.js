const express = require("express");
const AWS = require("aws-sdk");
const uuid = require("uuid");

// import express, { json } from 'express';
// import { DynamoDB } from 'aws-sdk';
// import uuid from 'uuid';

const app = express();
app.use(express.json());

const dynamodb = new AWS.DynamoDB.DocumentClient({
  region: "local",
  endpoint: "http://localhost:8000",
  accessKeyId: "AKID",
  secretAccessKey: "SECRET_KEY",
  sessionToken: "TOKEN",
});

const tableName = "MyTable";

async function main() {
  // Delete table if it exists
  dynamodb.service
    .deleteTable({ TableName: tableName })
    .promise()
    .catch((err) => {
      if (err && err.code !== "ResourceNotFoundException") {
        console.log(err);
      } else {
        throw err;
      }
    })
    .then(async () => {
      console.log("Deleted Table" + tableName);

      // Create table if it doesn't exist
      await dynamodb.service.createTable(
        {
          TableName: tableName,
          AttributeDefinitions: [
            {
              AttributeName: "Id",
              AttributeType: "S",
            },
          ],
          KeySchema: [
            {
              AttributeName: "Id",
              KeyType: "HASH",
            },
          ],
          ProvisionedThroughput: {
            ReadCapacityUnits: 5000,
            WriteCapacityUnits: 5000,
          },
        },
        (err, data) => {
          if (err) {
            console.log(err);
          } else {
            console.log(`Created table ${tableName}`);
          }
        }
      );
    });

  app.post("/create", (req, res) => {
    const { name, email, attributes } = req.body;
    const generatedId = uuid.v4().toString();

    // attributes is an object with key-value pairs, convert it to a ddb map
    for (const key in attributes) {
      attributes[key] = {
        S: attributes[key],
      };
    }
    const params = {
      TableName: tableName,
      Item: {
        Id: generatedId,
        Name: name,
        Email: email,
        Attributes: attributes,
      },
    };
    dynamodb.put(params, (err, data) => {
      if (err) {
        console.error(err);
        res.status(500).send(`Failed to store item: ${err}`);
      } else {
        res.status(201).send(`Item created with id: ${generatedId}`);
      }
    });
  });

  app.get("/list", (req, res) => {
    const limit = 1000;
    let lastEvaluatedKey;
    const continuationToken = req.query.continuationToken;
    if (continuationToken) {
      lastEvaluatedKey = {
        Id: continuationToken,
      };
    }
    const params = {
      TableName: tableName,
      Limit: limit,
      ExclusiveStartKey: lastEvaluatedKey,
    };
    dynamodb.scan(params, (err, data) => {
      if (err) {
        res.status(500).send(`Failed to list items: ${err}`);
      } else {
        const listResp = {
          items: [],
        };
        data.Items.forEach((item) => {
          const attributes = {};
          for (const [key, value] of Object.entries(item.Attributes)) {
            attributes[key] = value.S;
          }
          listResp.items.push({
            Id: item.Id,
            Name: item.Name,
            Email: item.Email,
            Attributes: attributes,
          });
        });
        if (data.LastEvaluatedKey) {
          listResp.continuationToken = data.LastEvaluatedKey.Id.S;
        }
        res.json(listResp);
      }
    });
  });

  app.listen(8081, () => {
    console.log("Server listening on port 8081");
  });
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
