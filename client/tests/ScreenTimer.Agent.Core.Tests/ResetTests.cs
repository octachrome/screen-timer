using ScreenTimer.Agent.Core.Engine;
using ScreenTimer.Agent.Core.Models;

namespace ScreenTimer.Agent.Core.Tests;

public class ResetTests
{
    private static readonly DateTimeOffset Yesterday = new(2026, 3, 31, 12, 0, 0, TimeSpan.FromHours(0));
    private static readonly DateTimeOffset Today = new(2026, 4, 1, 12, 0, 0, TimeSpan.FromHours(0));

    private static AgentState CreateStateWithYesterdayUsage()
    {
        var state = new AgentState
        {
            CurrentDate = Yesterday.LocalDateTime.Date.ToString("yyyy-MM-dd"),
            LastTickTime = Yesterday,
            LastForegroundExe = null,
            CurrentRules = new List<GroupRule>
            {
                new() { Name = "game.exe", Processes = new List<string> { "game.exe" }, DailyBudgetMinutes = 60 },
                new() { Name = "social.exe", Processes = new List<string> { "social.exe" }, DailyBudgetMinutes = 30 },
            },
            Apps = new Dictionary<string, AppUsageState>(StringComparer.OrdinalIgnoreCase)
            {
                ["game.exe"] = new AppUsageState
                {
                    UsedTodaySeconds = 3000,
                    PendingUploadSeconds = 120,
                },
                ["social.exe"] = new AppUsageState
                {
                    UsedTodaySeconds = 1800,
                    PendingUploadSeconds = 45,
                },
            },
            GroupUsage = new Dictionary<string, GroupUsageState>(StringComparer.OrdinalIgnoreCase)
            {
                ["game.exe"] = new GroupUsageState
                {
                    Sent10Min = true,
                    Sent5Min = true,
                    Sent1Min = false,
                    Exhausted = false,
                },
                ["social.exe"] = new GroupUsageState
                {
                    Sent10Min = true,
                    Sent5Min = true,
                    Sent1Min = true,
                    Exhausted = true,
                },
            },
        };
        return state;
    }

    [Fact]
    public void DateChange_ClearsUsedTodaySeconds_ForAllApps()
    {
        var state = CreateStateWithYesterdayUsage();
        var sample = new ForegroundSample("game.exe", Today);

        var result = AgentEngine.Tick(state, sample, null);

        Assert.Equal(0, result.UpdatedState.Apps["game.exe"].UsedTodaySeconds);
        Assert.Equal(0, result.UpdatedState.Apps["social.exe"].UsedTodaySeconds);
    }

    [Fact]
    public void DateChange_ClearsNotificationFlags_ForAllApps()
    {
        var state = CreateStateWithYesterdayUsage();
        var sample = new ForegroundSample("game.exe", Today);

        var result = AgentEngine.Tick(state, sample, null);

        var game = result.UpdatedState.GroupUsage["game.exe"];
        Assert.False(game.Sent10Min);
        Assert.False(game.Sent5Min);
        Assert.False(game.Sent1Min);

        var social = result.UpdatedState.GroupUsage["social.exe"];
        Assert.False(social.Sent10Min);
        Assert.False(social.Sent5Min);
        Assert.False(social.Sent1Min);
    }

    [Fact]
    public void DateChange_ClearsExhaustedFlag_ForAllApps()
    {
        var state = CreateStateWithYesterdayUsage();
        var sample = new ForegroundSample("social.exe", Today);

        var result = AgentEngine.Tick(state, sample, null);

        Assert.False(result.UpdatedState.GroupUsage["game.exe"].Exhausted);
        Assert.False(result.UpdatedState.GroupUsage["social.exe"].Exhausted);
    }

    [Fact]
    public void DateChange_UpdatesCurrentDate()
    {
        var state = CreateStateWithYesterdayUsage();
        var sample = new ForegroundSample("game.exe", Today);

        var result = AgentEngine.Tick(state, sample, null);

        Assert.Equal(Today.LocalDateTime.Date.ToString("yyyy-MM-dd"), result.UpdatedState.CurrentDate);
    }

    [Fact]
    public void DateChange_DoesNotClear_PendingUploadSeconds()
    {
        var state = CreateStateWithYesterdayUsage();
        var sample = new ForegroundSample("game.exe", Today);

        var result = AgentEngine.Tick(state, sample, null);

        Assert.Equal(120, result.UpdatedState.Apps["game.exe"].PendingUploadSeconds);
        Assert.Equal(45, result.UpdatedState.Apps["social.exe"].PendingUploadSeconds);
    }
}
