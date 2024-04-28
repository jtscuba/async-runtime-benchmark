using Amazon.DynamoDBv2.Model;
using AsyncRunTimeBenchmark.Database;
using Microsoft.AspNetCore.Mvc;

namespace AsyncRunTimeBenchmark.Controllers
{
	[ApiController]
	[Route("[controller]")]
	public class AsyncBenchmarkingController : ControllerBase
	{
		IDynamoDBHandler _dynamoDBHandler;
		private readonly ILogger<AsyncBenchmarkingController> _logger;

		public AsyncBenchmarkingController(ILogger<AsyncBenchmarkingController> logger, IDynamoDBHandler dynamoDBHandler)
		{
			_logger = logger;
			_dynamoDBHandler = dynamoDBHandler;
		}

		[HttpGet("tables", Name = "GetDataBaseTables")]
		public ListTablesResponse Get()
		{
			return _dynamoDBHandler.ListTablesAsync().Result;
		}

		[HttpGet("users", Name = "GetUserTableRows")]
		public async Task<IActionResult> GetUsers()
		{
			var items = await _dynamoDBHandler.GetItemsAsync();
			return Ok(items);
		}

		[HttpPut]
		public async Task<IActionResult> Put([FromBody] User user)
		{
			await _dynamoDBHandler.PutUserAsync(user);
			return Ok();
		}
	}
}
