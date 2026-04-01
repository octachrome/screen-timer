using ScreenTimer.Agent.Core.Dtos;
using ScreenTimer.Agent.Core.Interfaces;
using ScreenTimer.Agent.Core.Models;

namespace ScreenTimer.Agent.IntegrationTests;

public class FakeClock : IClock
{
    private DateTimeOffset _now = new(2026, 4, 1, 10, 0, 0, TimeSpan.Zero);
    public DateTimeOffset Now => _now;
    public void Advance(TimeSpan duration) => _now = _now.Add(duration);
}

public class FakeProbe : IForegroundWindowProbe
{
    public string? CurrentExe { get; set; }
    private readonly FakeClock _clock;
    public FakeProbe(FakeClock clock) => _clock = clock;
    public ForegroundSample Sample() => new(CurrentExe, _clock.Now);
}

public class FakeApiClient : IAgentApiClient
{
    public List<AppConfigDto> Configs { get; set; } = new();
    public List<UsagePushDto> PushedUsage { get; } = new();
    public bool ShouldFail { get; set; }

    public Task<List<AppConfigDto>> GetConfigAsync(CancellationToken ct = default)
    {
        if (ShouldFail) throw new HttpRequestException("Fake network error");
        return Task.FromResult(Configs);
    }

    public Task PushUsageAsync(UsagePushDto push, CancellationToken ct = default)
    {
        if (ShouldFail) throw new HttpRequestException("Fake network error");
        PushedUsage.Add(push);
        return Task.CompletedTask;
    }
}

public class FakeNotificationSink : INotificationSink
{
    public List<(string ExeName, int RemainingMinutes)> Toasts { get; } = new();
    public void ShowToast(string exeName, int remainingMinutes) => Toasts.Add((exeName, remainingMinutes));
}

public class FakeProcessController : IProcessController
{
    public List<string> ClosedProcesses { get; } = new();
    public Task ForceCloseAsync(string exeName)
    {
        ClosedProcesses.Add(exeName);
        return Task.CompletedTask;
    }
}

public class FakeStateStore : IStateStore
{
    public AgentState? StoredState { get; set; }
    public int SaveCount { get; private set; }
    public AgentState? Load() => StoredState;
    public void Save(AgentState state)
    {
        StoredState = state;
        SaveCount++;
    }
}
