using ScreenTimer.Agent.Core.Engine;
using ScreenTimer.Agent.Core.Models;

namespace ScreenTimer.Agent.Core.Tests;

public class TrackingTests
{
    private static readonly DateTimeOffset BaseTime = new(2026, 4, 1, 10, 0, 0, TimeSpan.Zero);

    private static AgentState CreateState(string date, params (string exe, int budgetMinutes)[] apps)
    {
        var state = new AgentState { CurrentDate = date, LastUsageFlushTime = BaseTime };
        var rules = new List<GroupRule>();
        foreach (var (exe, budget) in apps)
        {
            state.Apps[exe] = new AppUsageState();
            state.GroupUsage[exe] = new GroupUsageState();
            rules.Add(new GroupRule { Name = exe, Processes = new List<string> { exe }, DailyBudgetMinutes = budget });
        }
        state.CurrentRules = rules;
        return state;
    }

    private static ForegroundSample Sample(string? exe, DateTimeOffset time) => new(exe, time);

    [Fact]
    public void FirstTick_DoesNotAttributeTime()
    {
        var state = CreateState("2026-04-01", ("game.exe", 120));

        var result = AgentEngine.Tick(state, Sample("game.exe", BaseTime), null);

        Assert.Equal(0, result.UpdatedState.Apps["game.exe"].UsedTodaySeconds);
    }

    [Fact]
    public void SecondTick_AttributesElapsedTimeToPreviousForegroundExe()
    {
        var state = CreateState("2026-04-01", ("game.exe", 120));

        // First tick: game.exe is foreground
        var r1 = AgentEngine.Tick(state, Sample("game.exe", BaseTime), null);

        // Second tick: 1 second later, game.exe still foreground
        var r2 = AgentEngine.Tick(r1.UpdatedState, Sample("game.exe", BaseTime.AddSeconds(1)), null);

        Assert.Equal(1.0, r2.UpdatedState.Apps["game.exe"].UsedTodaySeconds);
    }

    [Fact]
    public void UntrackedApp_DoesNotAccumulateTime()
    {
        var state = CreateState("2026-04-01", ("game.exe", 120));

        // First tick: untracked app is foreground
        var r1 = AgentEngine.Tick(state, Sample("explorer.exe", BaseTime), null);

        // Second tick: still untracked
        var r2 = AgentEngine.Tick(r1.UpdatedState, Sample("explorer.exe", BaseTime.AddSeconds(5)), null);

        Assert.Equal(0, r2.UpdatedState.Apps["game.exe"].UsedTodaySeconds);
    }

    [Fact]
    public void SwitchingForeground_AttributesTimeToPreviousApp()
    {
        var state = CreateState("2026-04-01", ("game.exe", 120), ("browser.exe", 120));

        // Tick 1: game.exe foreground
        var r1 = AgentEngine.Tick(state, Sample("game.exe", BaseTime), null);

        // Tick 2: 3s later, switch to browser.exe — attributes 3s to game.exe
        var r2 = AgentEngine.Tick(r1.UpdatedState, Sample("browser.exe", BaseTime.AddSeconds(3)), null);

        Assert.Equal(3.0, r2.UpdatedState.Apps["game.exe"].UsedTodaySeconds);
        Assert.Equal(0.0, r2.UpdatedState.Apps["browser.exe"].UsedTodaySeconds);

        // Tick 3: 2s later, still browser.exe — attributes 2s to browser.exe
        var r3 = AgentEngine.Tick(r2.UpdatedState, Sample("browser.exe", BaseTime.AddSeconds(5)), null);

        Assert.Equal(3.0, r3.UpdatedState.Apps["game.exe"].UsedTodaySeconds);
        Assert.Equal(2.0, r3.UpdatedState.Apps["browser.exe"].UsedTodaySeconds);
    }

    [Fact]
    public void NullForeground_DoesNotCountTime()
    {
        var state = CreateState("2026-04-01", ("game.exe", 120));

        // Tick 1: game.exe foreground
        var r1 = AgentEngine.Tick(state, Sample("game.exe", BaseTime), null);

        // Tick 2: 2s later, game.exe → 2s attributed
        var r2 = AgentEngine.Tick(r1.UpdatedState, Sample("game.exe", BaseTime.AddSeconds(2)), null);
        Assert.Equal(2.0, r2.UpdatedState.Apps["game.exe"].UsedTodaySeconds);

        // Tick 3: 3s later, null foreground — still attributes 3s to game.exe (previous)
        var r3 = AgentEngine.Tick(r2.UpdatedState, Sample(null, BaseTime.AddSeconds(5)), null);
        Assert.Equal(5.0, r3.UpdatedState.Apps["game.exe"].UsedTodaySeconds);

        // Tick 4: 4s later, null foreground — no previous tracked exe, no attribution
        var r4 = AgentEngine.Tick(r3.UpdatedState, Sample(null, BaseTime.AddSeconds(9)), null);
        Assert.Equal(5.0, r4.UpdatedState.Apps["game.exe"].UsedTodaySeconds);
    }

    [Fact]
    public void UntrackedForeground_DoesNotCountTime()
    {
        var state = CreateState("2026-04-01", ("game.exe", 120));

        // Tick 1: untracked app foreground
        var r1 = AgentEngine.Tick(state, Sample("notepad.exe", BaseTime), null);

        // Tick 2: 5s later, game.exe foreground — no time attributed (previous was untracked)
        var r2 = AgentEngine.Tick(r1.UpdatedState, Sample("game.exe", BaseTime.AddSeconds(5)), null);

        Assert.Equal(0, r2.UpdatedState.Apps["game.exe"].UsedTodaySeconds);
    }

    [Fact]
    public void VariableTickIntervals_AttributesCorrectElapsedTime()
    {
        var state = CreateState("2026-04-01", ("game.exe", 120));

        // Tick 1: game.exe foreground at t=0
        var r1 = AgentEngine.Tick(state, Sample("game.exe", BaseTime), null);

        // Tick 2: 1s later
        var r2 = AgentEngine.Tick(r1.UpdatedState, Sample("game.exe", BaseTime.AddSeconds(1)), null);
        Assert.Equal(1.0, r2.UpdatedState.Apps["game.exe"].UsedTodaySeconds);

        // Tick 3: 2.5s later (variable interval)
        var r3 = AgentEngine.Tick(r2.UpdatedState, Sample("game.exe", BaseTime.AddSeconds(3.5)), null);
        Assert.Equal(3.5, r3.UpdatedState.Apps["game.exe"].UsedTodaySeconds);

        // Tick 4: 0.5s later
        var r4 = AgentEngine.Tick(r3.UpdatedState, Sample("game.exe", BaseTime.AddSeconds(4)), null);
        Assert.Equal(4.0, r4.UpdatedState.Apps["game.exe"].UsedTodaySeconds);
    }

    [Fact]
    public void PendingUploadSeconds_AccumulatesAlongsideUsedToday()
    {
        var state = CreateState("2026-04-01", ("game.exe", 120));

        var r1 = AgentEngine.Tick(state, Sample("game.exe", BaseTime), null);
        var r2 = AgentEngine.Tick(r1.UpdatedState, Sample("game.exe", BaseTime.AddSeconds(3)), null);

        Assert.Equal(3.0, r2.UpdatedState.Apps["game.exe"].UsedTodaySeconds);
        Assert.Equal(3.0, r2.UpdatedState.Apps["game.exe"].PendingUploadSeconds);
    }
}
