using ScreenTimer.Agent.Core.Engine;
using ScreenTimer.Agent.Core.Models;

namespace ScreenTimer.Agent.Core.Tests;

public class NotificationTests
{
    private static readonly DateTimeOffset BaseTime = new(2025, 6, 15, 12, 0, 0, TimeSpan.Zero);

    private static AgentState CreateState(string date, params (string exe, int budgetMinutes, double usedSeconds)[] apps)
    {
        var state = new AgentState { CurrentDate = date };
        var rules = new List<GroupRule>();
        foreach (var (exe, budget, used) in apps)
        {
            state.Apps[exe] = new AppUsageState { UsedTodaySeconds = used };
            state.GroupUsage[exe] = new GroupUsageState();
            rules.Add(new GroupRule { Name = exe, Processes = new List<string> { exe }, DailyBudgetMinutes = budget });
        }
        state.CurrentRules = rules;
        state.LastUsageFlushTime = BaseTime;
        return state;
    }

    private static ForegroundSample Sample(string? exe, DateTimeOffset? time = null)
        => new(exe, time ?? BaseTime);

    [Fact]
    public void Fires_10min_toast_when_remaining_crosses_threshold()
    {
        // Budget = 20 min (1200s). Used = 590s. After 1s tick, used = 591s, remaining = 609s → no toast.
        // Instead: used = 1190s. After 1s tick, used = 1191s, remaining = 9s? No — we want exactly 10min.
        // Budget = 20 min (1200s). Used = 599s. After 1s tick, used = 600s, remaining = 600s = 10 min. ≤600 → fires.
        var state = CreateState("2025-06-15", ("game.exe", 20, 599));
        state.LastTickTime = BaseTime;
        state.LastForegroundExe = "game.exe";

        var result = AgentEngine.Tick(state, Sample("game.exe", BaseTime.AddSeconds(1)), null);

        var toast = Assert.Single(result.Commands.OfType<ShowToastCommand>());
        Assert.Equal("game.exe", toast.Label);
        Assert.Equal(10, toast.RemainingMinutes);
    }

    [Fact]
    public void Fires_5min_toast_when_remaining_crosses_threshold()
    {
        // Budget = 20 min (1200s). Used = 899s. After 1s tick, used = 900s, remaining = 300s = 5 min.
        var state = CreateState("2025-06-15", ("game.exe", 20, 899));
        state.LastTickTime = BaseTime;
        state.LastForegroundExe = "game.exe";
        state.GroupUsage["game.exe"].Sent10Min = true; // already sent

        var result = AgentEngine.Tick(state, Sample("game.exe", BaseTime.AddSeconds(1)), null);

        var toast = Assert.Single(result.Commands.OfType<ShowToastCommand>());
        Assert.Equal(5, toast.RemainingMinutes);
    }

    [Fact]
    public void Fires_1min_toast_when_remaining_crosses_threshold()
    {
        // Budget = 20 min (1200s). Used = 1139s. After 1s tick, used = 1140s, remaining = 60s = 1 min.
        var state = CreateState("2025-06-15", ("game.exe", 20, 1139));
        state.LastTickTime = BaseTime;
        state.LastForegroundExe = "game.exe";
        state.GroupUsage["game.exe"].Sent10Min = true;
        state.GroupUsage["game.exe"].Sent5Min = true;

        var result = AgentEngine.Tick(state, Sample("game.exe", BaseTime.AddSeconds(1)), null);

        var toast = Assert.Single(result.Commands.OfType<ShowToastCommand>());
        Assert.Equal(1, toast.RemainingMinutes);
    }

    [Fact]
    public void Does_not_refire_notification_on_subsequent_tick()
    {
        // Budget = 20 min (1200s). Used = 600s, remaining = 600s = 10 min. Already sent.
        var state = CreateState("2025-06-15", ("game.exe", 20, 600));
        state.LastTickTime = BaseTime;
        state.LastForegroundExe = "game.exe";
        state.GroupUsage["game.exe"].Sent10Min = true;

        var result = AgentEngine.Tick(state, Sample("game.exe", BaseTime.AddSeconds(1)), null);

        Assert.Empty(result.Commands.OfType<ShowToastCommand>());
    }

    [Fact]
    public void Fires_all_crossed_thresholds_when_budget_already_past_on_first_tick()
    {
        // Budget = 20 min (1200s). Used = 960s, remaining = 240s = 4 min.
        // Both 10-min and 5-min thresholds are already crossed, so engine fires both.
        var state = CreateState("2025-06-15", ("game.exe", 20, 960));
        state.LastTickTime = BaseTime;
        state.LastForegroundExe = "game.exe";

        var result = AgentEngine.Tick(state, Sample("game.exe", BaseTime.AddSeconds(1)), null);

        var toasts = result.Commands.OfType<ShowToastCommand>().ToList();
        // Engine fires all unfired thresholds that are currently crossed
        Assert.Contains(toasts, t => t.RemainingMinutes == 10);
        Assert.Contains(toasts, t => t.RemainingMinutes == 5);
    }

    [Fact]
    public void Multiple_thresholds_crossed_in_single_tick()
    {
        // Budget = 20 min (1200s). Used = 500s. Large elapsed = 650s. After tick, used = 1150s, remaining = 50s.
        // Crosses 10min, 5min, and 1min thresholds all at once.
        var state = CreateState("2025-06-15", ("game.exe", 20, 500));
        state.LastTickTime = BaseTime;
        state.LastForegroundExe = "game.exe";

        var result = AgentEngine.Tick(state, Sample("game.exe", BaseTime.AddSeconds(650)), null);

        var toasts = result.Commands.OfType<ShowToastCommand>().ToList();
        Assert.Equal(3, toasts.Count);
        Assert.Contains(toasts, t => t.RemainingMinutes == 10);
        Assert.Contains(toasts, t => t.RemainingMinutes == 5);
        Assert.Contains(toasts, t => t.RemainingMinutes == 1);
    }

    [Fact]
    public void No_toast_when_remaining_is_zero_or_negative()
    {
        // Budget = 10 min (600s). Used = 599s. After 2s tick, used = 601s, remaining = -1s.
        // remaining <= threshold but remaining <= 0 → no toast.
        var state = CreateState("2025-06-15", ("game.exe", 10, 599));
        state.LastTickTime = BaseTime;
        state.LastForegroundExe = "game.exe";

        var result = AgentEngine.Tick(state, Sample("game.exe", BaseTime.AddSeconds(2)), null);

        Assert.Empty(result.Commands.OfType<ShowToastCommand>());
    }
}
