using Microsoft.Extensions.Logging.Abstractions;
using ScreenTimer.Agent.Core.Dtos;
using ScreenTimer.Agent.Core.Models;
using ScreenTimer.Agent.Host;

namespace ScreenTimer.Agent.IntegrationTests;

public class WorkerIntegrationTests
{
    private static readonly DateTimeOffset BaseTime = new(2026, 4, 1, 10, 0, 0, TimeSpan.Zero);

    private readonly FakeClock _clock = new();
    private readonly FakeProbe _probe;
    private readonly FakeApiClient _apiClient = new();
    private readonly FakeNotificationSink _notifications = new();
    private readonly FakeProcessController _processController = new();
    private readonly FakeStateStore _stateStore = new();

    public WorkerIntegrationTests()
    {
        _probe = new FakeProbe(_clock);
    }

    private AgentWorker CreateWorker() =>
        new(
            _probe,
            _apiClient,
            _notifications,
            _processController,
            _stateStore,
            _clock,
            NullLogger<AgentWorker>.Instance);

    private async Task RunWorkerForAsync(AgentWorker worker, TimeSpan duration)
    {
        using var cts = new CancellationTokenSource();
        await worker.StartAsync(cts.Token);
        await Task.Delay(duration);
        cts.Cancel();
        await worker.StopAsync(CancellationToken.None);
    }

    [Fact]
    public async Task Worker_Loads_Persisted_State_On_Startup()
    {
        var state = new AgentState
        {
            CurrentDate = "2026-04-01",
            LastForegroundExe = "game.exe",
            LastTickTime = BaseTime,
            Apps = new(StringComparer.OrdinalIgnoreCase)
            {
                ["game.exe"] = new AppUsageState { UsedTodaySeconds = 42.0 }
            },
            CurrentRules = new() { new GroupRule { Name = "game.exe", Processes = new List<string> { "game.exe" }, DailyBudgetMinutes = 120 } }
        };
        _stateStore.StoredState = state;

        _apiClient.Configs = new()
        {
            new GroupConfigDto { Name = "game.exe", Processes = new List<string> { "game.exe" }, DailyBudgetMinutes = 120 }
        };
        _probe.CurrentExe = "game.exe";

        var worker = CreateWorker();
        await RunWorkerForAsync(worker, TimeSpan.FromSeconds(2));

        // The worker loaded state, then ticked and persisted; the usage should include
        // the original 42s plus whatever accumulated during the run.
        Assert.NotNull(_stateStore.StoredState);
        Assert.True(_stateStore.StoredState!.Apps.ContainsKey("game.exe"));
        Assert.True(_stateStore.StoredState.Apps["game.exe"].UsedTodaySeconds >= 42.0);
    }

    [Fact]
    public async Task Worker_Polls_Config_And_Tracks_Usage()
    {
        _apiClient.Configs = new()
        {
            new GroupConfigDto { Name = "game.exe", Processes = new List<string> { "game.exe" }, DailyBudgetMinutes = 120 }
        };
        _probe.CurrentExe = "game.exe";

        var worker = CreateWorker();

        // Advance clock past the usage flush interval (15s) after a short real delay
        // so the engine sees enough elapsed time to trigger a flush.
        using var cts = new CancellationTokenSource();
        await worker.StartAsync(cts.Token);

        // Let the first tick run and pick up config
        await Task.Delay(TimeSpan.FromSeconds(1.5));

        // Jump the fake clock forward past the 15s flush interval
        _clock.Advance(TimeSpan.FromSeconds(16));

        // Let another tick run so it sees the elapsed time and flushes
        await Task.Delay(TimeSpan.FromSeconds(1.5));

        cts.Cancel();
        await worker.StopAsync(CancellationToken.None);

        Assert.True(_apiClient.PushedUsage.Count > 0,
            "Expected at least one usage push after advancing clock past flush interval");
        var pushed = _apiClient.PushedUsage[0];
        Assert.Contains(pushed.Usage, u => u.ExeName == "game.exe");
    }

    [Fact]
    public async Task Worker_Shows_Toast_At_Threshold()
    {
        _apiClient.Configs = new()
        {
            new GroupConfigDto { Name = "game.exe", Processes = new List<string> { "game.exe" }, DailyBudgetMinutes = 1 }
        };
        _probe.CurrentExe = "game.exe";

        // Preload state with usage just under the 1-minute warning threshold.
        // Budget = 60s, warning at remaining <= 60s, so at 0s used it's already at 60s remaining.
        // The 1-min toast fires when remaining <= 60 and > 0.
        // Use 10-min budget and preload near 10-min threshold instead.
        _apiClient.Configs = new()
        {
            new GroupConfigDto { Name = "game.exe", Processes = new List<string> { "game.exe" }, DailyBudgetMinutes = 1 }
        };

        // Preload state with 50s used out of 60s budget so remaining = 10s.
        // The 1-minute warning fires when remaining <= 60s, which is already true.
        // So we need the state to have Sent1Min = false but UsedToday = 50s.
        _stateStore.StoredState = new AgentState
        {
            CurrentDate = "2026-04-01",
            LastTickTime = BaseTime,
            LastForegroundExe = "game.exe",
            LastUsageFlushTime = BaseTime,
            Apps = new(StringComparer.OrdinalIgnoreCase)
            {
                ["game.exe"] = new AppUsageState { UsedTodaySeconds = 50.0 }
            },
            GroupUsage = new(StringComparer.OrdinalIgnoreCase)
            {
                ["game.exe"] = new GroupUsageState()
            },
            CurrentRules = new() { new GroupRule { Name = "game.exe", Processes = new List<string> { "game.exe" }, DailyBudgetMinutes = 1 } }
        };

        var worker = CreateWorker();
        await RunWorkerForAsync(worker, TimeSpan.FromSeconds(2));

        Assert.Contains(_notifications.Toasts, t => t.Label == "game.exe" && t.RemainingMinutes == 1);
    }

    [Fact]
    public async Task Worker_Force_Closes_Exhausted_App()
    {
        _apiClient.Configs = new()
        {
            new GroupConfigDto { Name = "game.exe", Processes = new List<string> { "game.exe" }, DailyBudgetMinutes = 1 }
        };
        _probe.CurrentExe = "game.exe";

        // Preload state with 59.5s used out of 60s budget.
        // After ~1s tick the remaining goes to ~0, triggering force close.
        _stateStore.StoredState = new AgentState
        {
            CurrentDate = "2026-04-01",
            LastTickTime = BaseTime,
            LastForegroundExe = "game.exe",
            LastUsageFlushTime = BaseTime,
            Apps = new(StringComparer.OrdinalIgnoreCase)
            {
                ["game.exe"] = new AppUsageState
                {
                    UsedTodaySeconds = 59.5
                }
            },
            GroupUsage = new(StringComparer.OrdinalIgnoreCase)
            {
                ["game.exe"] = new GroupUsageState
                {
                    Sent10Min = true,
                    Sent5Min = true,
                    Sent1Min = true
                }
            },
            CurrentRules = new() { new GroupRule { Name = "game.exe", Processes = new List<string> { "game.exe" }, DailyBudgetMinutes = 1 } }
        };

        var worker = CreateWorker();

        using var cts = new CancellationTokenSource();
        await worker.StartAsync(cts.Token);

        // Advance clock so the engine sees time passing
        await Task.Delay(TimeSpan.FromSeconds(1.5));
        _clock.Advance(TimeSpan.FromSeconds(2));
        await Task.Delay(TimeSpan.FromSeconds(1.5));

        cts.Cancel();
        await worker.StopAsync(CancellationToken.None);

        Assert.Contains("game.exe", _processController.ClosedProcesses);
    }

    [Fact]
    public async Task Worker_Persists_State_On_Usage_Flush()
    {
        _apiClient.Configs = new()
        {
            new GroupConfigDto { Name = "game.exe", Processes = new List<string> { "game.exe" }, DailyBudgetMinutes = 120 }
        };
        _probe.CurrentExe = "game.exe";

        var worker = CreateWorker();

        using var cts = new CancellationTokenSource();
        await worker.StartAsync(cts.Token);

        // Let first tick pick up config and start tracking
        await Task.Delay(TimeSpan.FromSeconds(1.5));

        // Jump clock past the 15s flush interval
        _clock.Advance(TimeSpan.FromSeconds(16));

        // Let tick run to trigger flush + persist
        await Task.Delay(TimeSpan.FromSeconds(1.5));

        cts.Cancel();
        await worker.StopAsync(CancellationToken.None);

        Assert.True(_stateStore.SaveCount > 0, "Expected state to be persisted at least once after usage flush");
    }

    [Fact]
    public async Task Worker_Handles_Api_Failure_Gracefully()
    {
        _apiClient.ShouldFail = true;
        _probe.CurrentExe = "game.exe";

        var worker = CreateWorker();
        await RunWorkerForAsync(worker, TimeSpan.FromSeconds(2));

        // Worker should not crash — if we get here, it handled the failure gracefully.
        // Verify no usage was pushed (all attempts failed).
        Assert.Empty(_apiClient.PushedUsage);
    }
}
