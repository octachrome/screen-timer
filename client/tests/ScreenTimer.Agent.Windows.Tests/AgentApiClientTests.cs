using System.Net;
using System.Text;
using System.Text.Json;
using ScreenTimer.Agent.Core.Dtos;
using ScreenTimer.Agent.Windows.Http;

namespace ScreenTimer.Agent.Windows.Tests;

public class AgentApiClientTests
{
    [Fact]
    public async Task GetConfigAsync_Deserializes_SnakeCaseJson()
    {
        var json = """
            {
                "apps": [
                    {"exe_name": "game.exe", "daily_budget_minutes": 60},
                    {"exe_name": "browser.exe", "daily_budget_minutes": 120}
                ]
            }
            """;

        var handler = new FakeHttpHandler(json, HttpStatusCode.OK);
        var client = CreateClient(handler);

        var response = await client.GetConfigAsync();

        Assert.Equal(2, response.Apps.Count);
        Assert.Equal("game.exe", response.Apps[0].ExeName);
        Assert.Equal(60, response.Apps[0].DailyBudgetMinutes);
        Assert.Equal("browser.exe", response.Apps[1].ExeName);
        Assert.Equal(120, response.Apps[1].DailyBudgetMinutes);
    }

    [Fact]
    public async Task GetConfigAsync_Returns_EmptyList_For_EmptyApps()
    {
        var handler = new FakeHttpHandler("""{"apps": []}""", HttpStatusCode.OK);
        var client = CreateClient(handler);

        var response = await client.GetConfigAsync();

        Assert.Empty(response.Apps);
    }

    [Fact]
    public async Task GetConfigAsync_Deserializes_TestPopupAt()
    {
        var json = """
            {
                "apps": [],
                "test_popup_at": "2025-06-15T12:00:00Z"
            }
            """;

        var handler = new FakeHttpHandler(json, HttpStatusCode.OK);
        var client = CreateClient(handler);

        var response = await client.GetConfigAsync();

        Assert.Equal("2025-06-15T12:00:00Z", response.TestPopupAt);
    }

    [Fact]
    public async Task GetConfigAsync_Throws_On_ErrorStatus()
    {
        var handler = new FakeHttpHandler("error", HttpStatusCode.InternalServerError);
        var client = CreateClient(handler);

        await Assert.ThrowsAsync<HttpRequestException>(() => client.GetConfigAsync());
    }

    [Fact]
    public async Task PushUsageAsync_Sends_SnakeCaseJson()
    {
        var handler = new FakeHttpHandler("""{"status":"ok"}""", HttpStatusCode.OK);
        var client = CreateClient(handler);

        var push = new UsagePushDto
        {
            Usage = new List<UsageReportDto>
            {
                new() { ExeName = "game.exe", Seconds = 120 },
                new() { ExeName = "browser.exe", Seconds = 45 }
            }
        };

        await client.PushUsageAsync(push);

        Assert.NotNull(handler.LastRequestBody);
        var doc = JsonDocument.Parse(handler.LastRequestBody);
        var root = doc.RootElement;

        Assert.True(root.TryGetProperty("usage", out var usageArr));
        Assert.Equal(2, usageArr.GetArrayLength());

        var first = usageArr[0];
        Assert.Equal("game.exe", first.GetProperty("exe_name").GetString());
        Assert.Equal(120, first.GetProperty("seconds").GetInt32());

        var second = usageArr[1];
        Assert.Equal("browser.exe", second.GetProperty("exe_name").GetString());
        Assert.Equal(45, second.GetProperty("seconds").GetInt32());
    }

    [Fact]
    public async Task PushUsageAsync_PostsToCorrectEndpoint()
    {
        var handler = new FakeHttpHandler("""{"status":"ok"}""", HttpStatusCode.OK);
        var client = CreateClient(handler);

        await client.PushUsageAsync(new UsagePushDto
        {
            Usage = [new() { ExeName = "game.exe", Seconds = 10 }]
        });

        Assert.Equal(HttpMethod.Post, handler.LastRequestMethod);
        Assert.Equal("/api/agent/usage", handler.LastRequestUri?.AbsolutePath);
    }

    [Fact]
    public async Task GetConfigAsync_GetsFromCorrectEndpoint()
    {
        var handler = new FakeHttpHandler("""{"apps": []}""", HttpStatusCode.OK);
        var client = CreateClient(handler);

        await client.GetConfigAsync();

        Assert.Equal(HttpMethod.Get, handler.LastRequestMethod);
        Assert.Equal("/api/agent/config", handler.LastRequestUri?.AbsolutePath);
    }

    [Fact]
    public async Task PushUsageAsync_Throws_On_ErrorStatus()
    {
        var handler = new FakeHttpHandler("""{"error":"bad request"}""", HttpStatusCode.BadRequest);
        var client = CreateClient(handler);

        await Assert.ThrowsAsync<HttpRequestException>(() =>
            client.PushUsageAsync(new UsagePushDto
            {
                Usage = [new() { ExeName = "x.exe", Seconds = 1 }]
            }));
    }

    private static AgentApiClient CreateClient(FakeHttpHandler handler)
    {
        var httpClient = new HttpClient(handler)
        {
            BaseAddress = new Uri("http://localhost:8080")
        };
        return new AgentApiClient(httpClient);
    }

    private sealed class FakeHttpHandler(string responseBody, HttpStatusCode statusCode) : HttpMessageHandler
    {
        public HttpMethod? LastRequestMethod { get; private set; }
        public Uri? LastRequestUri { get; private set; }
        public string? LastRequestBody { get; private set; }

        protected override async Task<HttpResponseMessage> SendAsync(
            HttpRequestMessage request, CancellationToken cancellationToken)
        {
            LastRequestMethod = request.Method;
            LastRequestUri = request.RequestUri;
            if (request.Content is not null)
                LastRequestBody = await request.Content.ReadAsStringAsync(cancellationToken);

            return new HttpResponseMessage(statusCode)
            {
                Content = new StringContent(responseBody, Encoding.UTF8, "application/json")
            };
        }
    }
}
