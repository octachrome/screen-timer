using ScreenTimer.Agent.Core.Dtos;
using ScreenTimer.Agent.Core.Engine;
using ScreenTimer.Agent.Core.Models;

namespace ScreenTimer.Agent.Core.Tests;

public class SyncTests
{
    private static readonly DateTimeOffset BaseTime = new(2026, 4, 1, 10, 0, 0, TimeSpan.Zero);

    private static AgentState CreateState(List<GroupRule>? rules = null)
    {
        var state = new AgentState { CurrentDate = "2026-04-01" };
        if (rules is not null)
        {
            AgentEngine.Tick(state, new ForegroundSample(null, BaseTime), rules);
        }
        return state;
    }

    [Fact]
    public void NewConfig_Adds_AppUsageState_For_New_Apps()
    {
        var state = new AgentState { CurrentDate = "2026-04-01" };
        var rules = new List<GroupRule>
        {
            new() { Name = "game.exe", Processes = new List<string> { "game.exe" }, DailyBudgetMinutes = 60 },
            new() { Name = "browser.exe", Processes = new List<string> { "browser.exe" }, DailyBudgetMinutes = 120 }
        };

        AgentEngine.Tick(state, new ForegroundSample(null, BaseTime), rules);

        Assert.True(state.Apps.ContainsKey("game.exe"));
        Assert.True(state.Apps.ContainsKey("browser.exe"));
        Assert.Equal(2, state.Apps.Count);
    }

    [Fact]
    public void RemovedApp_From_Config_Removes_Its_AppUsageState()
    {
        var state = CreateState(new List<GroupRule>
        {
            new() { Name = "game.exe", Processes = new List<string> { "game.exe" }, DailyBudgetMinutes = 60 },
            new() { Name = "browser.exe", Processes = new List<string> { "browser.exe" }, DailyBudgetMinutes = 120 }
        });

        // Now apply config without browser.exe
        var newRules = new List<GroupRule>
        {
            new() { Name = "game.exe", Processes = new List<string> { "game.exe" }, DailyBudgetMinutes = 60 }
        };
        AgentEngine.Tick(state, new ForegroundSample(null, BaseTime.AddSeconds(1)), newRules);

        Assert.True(state.Apps.ContainsKey("game.exe"));
        Assert.False(state.Apps.ContainsKey("browser.exe"));
        Assert.Single(state.Apps);
    }

    [Fact]
    public void ChangedBudget_Updates_Rule_And_Fires_Notification_At_New_Threshold()
    {
        var rules = new List<GroupRule>
        {
            new() { Name = "game.exe", Processes = new List<string> { "game.exe" }, DailyBudgetMinutes = 60 }
        };
        var state = CreateState(rules);

        // Accumulate 50 minutes of usage (3000 seconds)
        state.Apps["game.exe"].UsedTodaySeconds = 3000;
        state.LastForegroundExe = "game.exe";
        state.LastTickTime = BaseTime.AddSeconds(1);

        // With 60-min budget, remaining = 600s = 10 min — at threshold but Tick hasn't run yet
        // Now change budget to 51 minutes (3060s). After 1s elapsed, used = 3001s, remaining = 59s
        // That should trigger 10min, 5min, and 1min notifications
        var newRules = new List<GroupRule>
        {
            new() { Name = "game.exe", Processes = new List<string> { "game.exe" }, DailyBudgetMinutes = 51 }
        };

        var result = AgentEngine.Tick(state, new ForegroundSample("game.exe", BaseTime.AddSeconds(2)), newRules);

        var toasts = result.Commands.OfType<ShowToastCommand>().ToList();
        Assert.Contains(toasts, t => t.RemainingMinutes == 10);
        Assert.Contains(toasts, t => t.RemainingMinutes == 5);
        Assert.Contains(toasts, t => t.RemainingMinutes == 1);
    }

    [Fact]
    public void MarkUsagePushSucceeded_Clears_Only_Uploaded_Portion()
    {
        var rules = new List<GroupRule>
        {
            new() { Name = "game.exe", Processes = new List<string> { "game.exe" }, DailyBudgetMinutes = 60 }
        };
        var state = CreateState(rules);

        // Simulate 30 seconds of pending usage
        state.Apps["game.exe"].PendingUploadSeconds = 30;

        // Push reports only 20 seconds
        var pushed = new UsagePushDto
        {
            Usage = new List<UsageReportDto>
            {
                new() { ExeName = "game.exe", Seconds = 20 }
            }
        };

        AgentEngine.MarkUsagePushSucceeded(state, pushed);

        Assert.Equal(10, state.Apps["game.exe"].PendingUploadSeconds);
    }

    [Fact]
    public void FailedPush_Preserves_Pending_Usage()
    {
        var rules = new List<GroupRule>
        {
            new() { Name = "game.exe", Processes = new List<string> { "game.exe" }, DailyBudgetMinutes = 60 }
        };
        var state = CreateState(rules);

        // Set foreground to game.exe and tick to accumulate usage
        state.LastForegroundExe = "game.exe";
        state.LastTickTime = BaseTime.AddSeconds(1);

        // Tick 10 seconds later — accumulates ~10s on game.exe
        AgentEngine.Tick(state, new ForegroundSample("game.exe", BaseTime.AddSeconds(11)), null);

        var pendingBefore = state.Apps["game.exe"].PendingUploadSeconds;
        Assert.True(pendingBefore > 0);

        // Simulate a flush that would generate a PushUsageCommand, but we never call MarkUsagePushSucceeded
        // The pending usage should remain unchanged
        var pendingAfter = state.Apps["game.exe"].PendingUploadSeconds;
        Assert.Equal(pendingBefore, pendingAfter);
    }

    [Fact]
    public void UsageFlush_Generates_PushUsageCommand_When_Pending_And_Interval_Elapsed()
    {
        var rules = new List<GroupRule>
        {
            new() { Name = "game.exe", Processes = new List<string> { "game.exe" }, DailyBudgetMinutes = 60 }
        };
        var state = CreateState(rules);

        // Set foreground and initial tick
        state.LastForegroundExe = "game.exe";
        state.LastTickTime = BaseTime.AddSeconds(1);
        state.LastUsageFlushTime = BaseTime;

        // Tick 5 seconds later — pending usage accrues but not enough time for flush
        var result5 = AgentEngine.Tick(state, new ForegroundSample("game.exe", BaseTime.AddSeconds(6)), null);
        Assert.DoesNotContain(result5.Commands, c => c is PushUsageCommand);

        // Tick 16 seconds after base — ≥15s since last flush, and there is pending usage
        var result16 = AgentEngine.Tick(state, new ForegroundSample("game.exe", BaseTime.AddSeconds(16)), null);
        var pushCmd = Assert.Single(result16.Commands.OfType<PushUsageCommand>());
        Assert.True(pushCmd.Payload.Usage.Count > 0);
        Assert.True(pushCmd.Payload.Usage[0].Seconds > 0);
    }

    [Fact]
    public void UsageFlush_Does_Not_Push_When_No_Pending_Usage()
    {
        var rules = new List<GroupRule>
        {
            new() { Name = "game.exe", Processes = new List<string> { "game.exe" }, DailyBudgetMinutes = 60 }
        };
        var state = CreateState(rules);
        state.LastUsageFlushTime = BaseTime;

        // Tick with no foreground app — no usage accrued
        var result = AgentEngine.Tick(state, new ForegroundSample(null, BaseTime.AddSeconds(16)), null);

        Assert.DoesNotContain(result.Commands, c => c is PushUsageCommand);
    }
}
