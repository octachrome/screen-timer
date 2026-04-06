using ScreenTimer.Agent.Core.Engine;
using ScreenTimer.Agent.Core.Models;

namespace ScreenTimer.Agent.Core.Tests;

public class EnforcementTests
{
    private static readonly DateTimeOffset BaseTime = new(2025, 6, 15, 12, 0, 0, TimeSpan.Zero);

    private static AgentState CreateState(string exeName, int budgetMinutes, double usedSeconds)
    {
        var rule = new GroupRule { Name = exeName, Processes = new List<string> { exeName }, DailyBudgetMinutes = budgetMinutes };
        var state = new AgentState
        {
            CurrentDate = BaseTime.LocalDateTime.Date.ToString("yyyy-MM-dd"),
            CurrentRules = [rule],
            Apps = { [exeName] = new AppUsageState { UsedTodaySeconds = usedSeconds } },
            GroupUsage = { [exeName] = new GroupUsageState() },
            LastUsageFlushTime = BaseTime,
        };
        return state;
    }

    [Fact]
    public void Fires_ForceCloseCommand_when_budget_reaches_zero()
    {
        // Budget = 1 minute = 60s. Already used 55s.
        // Tick with "game.exe" as previous foreground for 5s → reaches exactly 60s.
        var state = CreateState("game.exe", budgetMinutes: 1, usedSeconds: 55);
        state.LastForegroundExe = "game.exe";
        state.LastTickTime = BaseTime;

        // After 5s the tracked app is still in foreground
        var sample = new ForegroundSample("game.exe", BaseTime.AddSeconds(5));
        var result = AgentEngine.Tick(state, sample, null);

        Assert.Contains(result.Commands, c => c is ForceCloseCommand { ExeName: "game.exe" });
    }

    [Fact]
    public void Refires_ForceCloseCommand_if_exhausted_app_reappears_in_foreground()
    {
        // Budget already fully exhausted
        var state = CreateState("game.exe", budgetMinutes: 1, usedSeconds: 60);
        state.LastForegroundExe = null;
        state.LastTickTime = BaseTime;
        state.GroupUsage["game.exe"].Exhausted = true;
        state.LastUsageFlushTime = BaseTime;

        // Exhausted app reappears in foreground
        var sample = new ForegroundSample("game.exe", BaseTime.AddSeconds(3));
        var result = AgentEngine.Tick(state, sample, null);

        Assert.Contains(result.Commands, c => c is ForceCloseCommand { ExeName: "game.exe" });
    }

    [Fact]
    public void Does_not_fire_ForceCloseCommand_for_untracked_app()
    {
        var state = CreateState("game.exe", budgetMinutes: 1, usedSeconds: 60);
        state.LastForegroundExe = null;
        state.LastTickTime = BaseTime;
        state.LastUsageFlushTime = BaseTime;

        // Untracked app in foreground
        var sample = new ForegroundSample("notepad.exe", BaseTime.AddSeconds(3));
        var result = AgentEngine.Tick(state, sample, null);

        Assert.DoesNotContain(result.Commands, c => c is ForceCloseCommand);
    }

    [Fact]
    public void Does_not_fire_ForceCloseCommand_when_exhausted_app_not_in_foreground()
    {
        // game.exe was previous foreground and budget will be exhausted,
        // but a different app is now in foreground
        var state = CreateState("game.exe", budgetMinutes: 1, usedSeconds: 58);
        state.LastForegroundExe = "game.exe";
        state.LastTickTime = BaseTime;
        state.LastUsageFlushTime = BaseTime;

        // Different (untracked) app now in foreground
        var sample = new ForegroundSample("notepad.exe", BaseTime.AddSeconds(5));
        var result = AgentEngine.Tick(state, sample, null);

        // Budget is exhausted (58 + 5 = 63 > 60), but notepad.exe is foreground
        Assert.DoesNotContain(result.Commands, c => c is ForceCloseCommand);
    }

    [Fact]
    public void Sets_Exhausted_flag_when_remaining_is_zero_or_less()
    {
        var state = CreateState("game.exe", budgetMinutes: 1, usedSeconds: 55);
        state.LastForegroundExe = "game.exe";
        state.LastTickTime = BaseTime;
        state.LastUsageFlushTime = BaseTime;

        var sample = new ForegroundSample("game.exe", BaseTime.AddSeconds(5));
        var result = AgentEngine.Tick(state, sample, null);

        Assert.True(result.UpdatedState.GroupUsage["game.exe"].Exhausted);
    }

    [Fact]
    public void ForceClose_Targets_Foreground_Exe_Not_Group_Name()
    {
        var rule = new GroupRule { Name = "Gaming", Processes = new List<string> { "game.exe" }, DailyBudgetMinutes = 1 };
        var state = new AgentState
        {
            CurrentDate = BaseTime.LocalDateTime.Date.ToString("yyyy-MM-dd"),
            CurrentRules = [rule],
            Apps = { ["game.exe"] = new AppUsageState { UsedTodaySeconds = 55 } },
            GroupUsage = { ["Gaming"] = new GroupUsageState() },
            LastUsageFlushTime = BaseTime,
            LastForegroundExe = "game.exe",
            LastTickTime = BaseTime,
        };

        var sample = new ForegroundSample("game.exe", BaseTime.AddSeconds(5));
        var result = AgentEngine.Tick(state, sample, null);

        var close = Assert.Single(result.Commands.OfType<ForceCloseCommand>());
        Assert.Equal("game.exe", close.ExeName);
    }
}
