using Amazon.DynamoDBv2.Model;

namespace AsyncRunTimeBenchmark.Database
{
	public interface IDynamoDBHandler
	{
		Task<ListTablesResponse> ListTablesAsync();
		Task PutUserAsync(User user);
		Task<List<Dictionary<string, AttributeValue>>> GetItemsAsync();
	}
}