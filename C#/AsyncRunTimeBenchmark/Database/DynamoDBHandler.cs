using Amazon.DynamoDBv2;
using Amazon.DynamoDBv2.Model;
using Amazon.Runtime;
namespace AsyncRunTimeBenchmark.Database
{
	public class User
	{
		public string Name { get; set; }
		public string Email { get; set; }
		public Dictionary<string, string> Attributes { get; set; }
	}

	public class DynamoDBHandler : IDynamoDBHandler
	{
		private readonly AmazonDynamoDBClient client;
		public DynamoDBHandler()
		{
			AmazonDynamoDBConfig config = new AmazonDynamoDBConfig
			{
				ServiceURL = "http://localhost:8000", // replace with your local DynamoDB instance URL
			};
			var credentials = new BasicAWSCredentials("dummy", "dummy");
			client = new AmazonDynamoDBClient(credentials, config);
			DeleteTable();
			CreateTable();
		}

		public async Task<ListTablesResponse> ListTablesAsync()
		{
			return await client.ListTablesAsync();
		}

		public async Task<List<Dictionary<string, AttributeValue>>> GetItemsAsync()
		{
			var request = new ScanRequest
			{
				TableName = "MyTable",
				Limit = 1000
			};

			var response = await client.ScanAsync(request);

			return response.Items;
		}

		public async Task PutUserAsync(User user)
		{
			var request = new PutItemRequest
			{
				TableName = "MyTable",
				Item = new Dictionary<string, AttributeValue>
			{
				{ "ID", new AttributeValue { S = Guid.NewGuid().ToString() } },
				{ "Name", new AttributeValue { S = user.Name } },
				{ "Email", new AttributeValue { S = user.Email } },
                // Assuming Attributes are also string values
                { "Attributes", new AttributeValue { M = ConvertToAttributeValues(user.Attributes) } }
			}
			};

			await client.PutItemAsync(request);
		}

		private Dictionary<string, AttributeValue> ConvertToAttributeValues(Dictionary<string, string> input)
		{
			var output = new Dictionary<string, AttributeValue>();
			foreach (var entry in input)
			{
				output[entry.Key] = new AttributeValue { S = entry.Value };
			}
			return output;
		}

		public void CreateTable()
		{
			var createTableRequest = new CreateTableRequest
			{
				TableName = "MyTable",
				AttributeDefinitions = new List<AttributeDefinition>
				{
					new AttributeDefinition
					{
						AttributeName = "ID",
						AttributeType = "S"
					}
				},
				KeySchema = new List<KeySchemaElement>
				{
					new KeySchemaElement
					{
						AttributeName = "ID",
						KeyType = "HASH"
					}
				},
				ProvisionedThroughput = new ProvisionedThroughput
				{
					ReadCapacityUnits = 5000,
					WriteCapacityUnits = 5000
				}
			};
			try
			{
				client.CreateTableAsync(createTableRequest);
			}
			catch (Exception)
			{
				throw;
			}
		}

		public void DeleteTable()
		{
			var deleteTableRequest = new DeleteTableRequest
			{
				TableName = "MyTable"
			};
			try
			{
				client.DeleteTableAsync(deleteTableRequest);
			}
			catch (Exception)
			{
				throw;
			}
		}
	}
}
