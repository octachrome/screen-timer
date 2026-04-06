using System.Diagnostics;
using System.Net;
using System.Net.Http.Json;
using System.Text.Json;
using ScreenTimer.Agent.Core.Dtos;
using ScreenTimer.Agent.Windows.Http;

namespace ScreenTimer.Agent.Windows.Tests;

/// <summary>
/// Cross-language contract smoke tests that start the real Go server
/// and verify C# DTO serialization matches the Go JSON contract.
/// Skipped automatically when Go is not installed.
/// </summary>
public class GoServerContractTests : IAsyncLifetime
{
    private Process? _serverProcess;
    private HttpClient? _rawHttp;
    private AgentApiClient? _agentClient;
    private int _port;
    private string? _tempDataDir;

    private static bool IsGoAvailable()
    {
        try
        {
            using var proc = Process.Start(new ProcessStartInfo("go", "version")
            {
                RedirectStandardOutput = true,
                RedirectStandardError = true,
                UseShellExecute = false,
                CreateNoWindow = true
            });
            proc?.WaitForExit(5000);
            return proc?.ExitCode == 0;
        }
        catch
        {
            return false;
        }
    }

    public async Task InitializeAsync()
    {
        if (!IsGoAvailable())
            return;

        _port = FindFreePort();
        _tempDataDir = Path.Combine(Path.GetTempPath(), "screen-timer-test-" + Guid.NewGuid().ToString("N"));
        Directory.CreateDirectory(_tempDataDir);
        var dataFile = Path.Combine(_tempDataDir, "data.json");

        var serverDir = Path.GetFullPath(Path.Combine(
            AppContext.BaseDirectory, "..", "..", "..", "..", "..", "..", "server"));

        _serverProcess = new Process
        {
            StartInfo = new ProcessStartInfo
            {
                FileName = "go",
                Arguments = "run ./cmd/server",
                WorkingDirectory = serverDir,
                UseShellExecute = false,
                RedirectStandardOutput = true,
                RedirectStandardError = true,
                CreateNoWindow = true,
                Environment = { ["PORT"] = _port.ToString(), ["DATA_FILE"] = dataFile }
            }
        };

        _serverProcess.Start();

        var baseUri = new Uri($"http://localhost:{_port}");
        _rawHttp = new HttpClient { BaseAddress = baseUri };
        _agentClient = new AgentApiClient(new HttpClient { BaseAddress = baseUri });

        await WaitForServerReady();
    }

    public Task DisposeAsync()
    {
        if (_serverProcess is { HasExited: false })
        {
            try
            {
                _serverProcess.Kill(entireProcessTree: true);
                _serverProcess.WaitForExit(3000);
            }
            catch { }
        }
        _serverProcess?.Dispose();
        _rawHttp?.Dispose();
        if (_tempDataDir is not null && Directory.Exists(_tempDataDir))
        {
            try { Directory.Delete(_tempDataDir, recursive: true); } catch { }
        }
        return Task.CompletedTask;
    }

    [Fact]
    public async Task GetConfig_Returns_SnakeCase_MatchingDto()
    {
        if (!IsGoAvailable()) return;

        // Add an app via the UI API
        await _rawHttp!.PostAsJsonAsync("/api/apps", new { name = "game.exe", processes = new[] { "game.exe" }, daily_budget_minutes = 60 });

        // Call agent config endpoint via C# client
        var configResponse = await _agentClient!.GetConfigAsync();

        Assert.Single(configResponse.Groups);
        Assert.Equal("game.exe", configResponse.Groups[0].Name);
        Assert.Equal(60, configResponse.Groups[0].DailyBudgetMinutes);
    }

    [Fact]
    public async Task PushUsage_Sends_SnakeCase_AcceptedByGoServer()
    {
        if (!IsGoAvailable()) return;

        // Add an app first
        await _rawHttp!.PostAsJsonAsync("/api/apps", new { name = "browser.exe", processes = new[] { "browser.exe" }, daily_budget_minutes = 120 });

        // Push usage via C# client
        var push = new UsagePushDto
        {
            Usage = [new UsageReportDto { ExeName = "browser.exe", Seconds = 300 }]
        };

        await _agentClient!.PushUsageAsync(push);

        // Verify usage was recorded by checking the usage summary
        var response = await _rawHttp!.GetAsync("/api/usage/today");
        response.EnsureSuccessStatusCode();
        var body = await response.Content.ReadAsStringAsync();
        var doc = JsonDocument.Parse(body);
        var apps = doc.RootElement.EnumerateArray().ToList();

        var browserApp = apps.First(a => a.GetProperty("name").GetString() == "browser.exe");
        Assert.Equal(5, browserApp.GetProperty("used_today_minutes").GetInt32());
    }

    [Fact]
    public async Task GetConfig_EmptyServer_Returns_EmptyArray()
    {
        if (!IsGoAvailable()) return;

        var configResponse = await _agentClient!.GetConfigAsync();

        Assert.Empty(configResponse.Groups);
    }

    private async Task WaitForServerReady()
    {
        using var cts = new CancellationTokenSource(TimeSpan.FromSeconds(30));
        while (!cts.IsCancellationRequested)
        {
            try
            {
                var response = await _rawHttp!.GetAsync("/healthz", cts.Token);
                if (response.StatusCode == HttpStatusCode.OK)
                    return;
            }
            catch (HttpRequestException) { }

            await Task.Delay(200, cts.Token);
        }

        throw new TimeoutException("Go server did not become ready within 30 seconds");
    }

    private static int FindFreePort()
    {
        using var listener = new System.Net.Sockets.TcpListener(IPAddress.Loopback, 0);
        listener.Start();
        var port = ((IPEndPoint)listener.LocalEndpoint).Port;
        listener.Stop();
        return port;
    }
}
